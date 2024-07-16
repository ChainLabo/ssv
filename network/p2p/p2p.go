package p2pv1

import (
	"context"
	"encoding/hex"
	"sync/atomic"
	"time"

	"github.com/cornelk/hashmap"
	"github.com/libp2p/go-libp2p/core/connmgr"
	connmgrcore "github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2pdiscbackoff "github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	"go.uber.org/zap"

	spectypes "github.com/ssvlabs/ssv-spec/types"
	"github.com/ssvlabs/ssv/logging"
	"github.com/ssvlabs/ssv/logging/fields"
	"github.com/ssvlabs/ssv/message/validation"
	"github.com/ssvlabs/ssv/network"
	"github.com/ssvlabs/ssv/network/commons"
	"github.com/ssvlabs/ssv/network/discovery"
	"github.com/ssvlabs/ssv/network/peers"
	"github.com/ssvlabs/ssv/network/peers/connections"
	"github.com/ssvlabs/ssv/network/records"
	"github.com/ssvlabs/ssv/network/streams"
	"github.com/ssvlabs/ssv/network/topics"
	operatordatastore "github.com/ssvlabs/ssv/operator/datastore"
	"github.com/ssvlabs/ssv/operator/keys"
	operatorstorage "github.com/ssvlabs/ssv/operator/storage"
	"github.com/ssvlabs/ssv/utils/async"
	"github.com/ssvlabs/ssv/utils/tasks"
)

// network states
const (
	stateInitializing int32 = 0
	stateClosing      int32 = 1
	stateClosed       int32 = 2
	stateReady        int32 = 10
)

const (
	connManagerGCInterval           = 3 * time.Minute
	connManagerGCTimeout            = time.Minute
	peersReportingInterval          = 60 * time.Second
	peerIdentitiesReportingInterval = 5 * time.Minute
	topicsReportingInterval         = 180 * time.Second
)

// PeersIndexProvider holds peers index instance
type PeersIndexProvider interface {
	PeersIndex() peers.Index
}

// HostProvider holds host instance
type HostProvider interface {
	Host() host.Host
}

// p2pNetwork implements network.P2PNetwork
type p2pNetwork struct {
	parentCtx context.Context
	ctx       context.Context
	cancel    context.CancelFunc

	interfaceLogger *zap.Logger // struct logger to log in interface methods that do not accept a logger
	cfg             *Config

	host         host.Host
	streamCtrl   streams.StreamController
	idx          peers.Index
	disc         discovery.Service
	topicsCtrl   topics.Controller
	msgRouter    network.MessageRouter
	msgResolver  topics.MsgPeersResolver
	msgValidator validation.MessageValidator
	connHandler  connections.ConnHandler
	connGater    connmgr.ConnectionGater
	metrics      Metrics

	state int32

	activeValidators *hashmap.Map[string, validatorStatus]

	backoffConnector *libp2pdiscbackoff.BackoffConnector
	subnets          []byte
	libConnManager   connmgrcore.ConnManager

	nodeStorage             operatorstorage.Storage
	operatorPKHashToPKCache *hashmap.Map[string, []byte] // used for metrics
	operatorSigner          keys.OperatorSigner
	operatorDataStore       operatordatastore.OperatorDataStore
}

// New creates a new p2p network
func New(logger *zap.Logger, cfg *Config, mr Metrics) network.P2PNetwork {
	ctx, cancel := context.WithCancel(cfg.Ctx)

	logger = logger.Named(logging.NameP2PNetwork)

	return &p2pNetwork{
		parentCtx:               cfg.Ctx,
		ctx:                     ctx,
		cancel:                  cancel,
		interfaceLogger:         logger,
		cfg:                     cfg,
		msgRouter:               cfg.Router,
		msgValidator:            cfg.MessageValidator,
		state:                   stateClosed,
		activeValidators:        hashmap.New[string, validatorStatus](),
		nodeStorage:             cfg.NodeStorage,
		operatorPKHashToPKCache: hashmap.New[string, []byte](),
		operatorSigner:          cfg.OperatorSigner,
		operatorDataStore:       cfg.OperatorDataStore,
		metrics:                 mr,
	}
}

// Host implements HostProvider
func (n *p2pNetwork) Host() host.Host {
	return n.host
}

// PeersIndex returns the peers index
func (n *p2pNetwork) PeersIndex() peers.Index {
	return n.idx
}

func (n *p2pNetwork) PeersByTopic() ([]peer.ID, map[string][]peer.ID) {
	var err error
	tpcs := n.topicsCtrl.Topics()
	peerz := make(map[string][]peer.ID, len(tpcs))
	for _, tpc := range tpcs {
		peerz[tpc], err = n.topicsCtrl.Peers(tpc)
		if err != nil {
			n.interfaceLogger.Error("Cant get peers from topics")
			return nil, nil
		}
	}
	allpeers, err := n.topicsCtrl.Peers("")
	if err != nil {
		n.interfaceLogger.Error("Cant all peers")
		return nil, nil
	}
	return allpeers, peerz
}

// Close implements io.Closer
func (n *p2pNetwork) Close() error {
	atomic.SwapInt32(&n.state, stateClosing)
	defer atomic.StoreInt32(&n.state, stateClosed)
	n.cancel()
	if err := n.libConnManager.Close(); err != nil {
		n.interfaceLogger.Warn("could not close discovery", zap.Error(err))
	}
	if err := n.disc.Close(); err != nil {
		n.interfaceLogger.Warn("could not close discovery", zap.Error(err))
	}
	if err := n.idx.Close(); err != nil {
		n.interfaceLogger.Warn("could not close index", zap.Error(err))
	}
	if err := n.topicsCtrl.Close(); err != nil {
		n.interfaceLogger.Warn("could not close topics controller", zap.Error(err))
	}
	return n.host.Close()
}

// Start starts the discovery service, garbage collector (peer index), and reporting.
func (n *p2pNetwork) Start(logger *zap.Logger) error {
	logger = logger.Named(logging.NameP2PNetwork)

	if atomic.SwapInt32(&n.state, stateReady) == stateReady {
		// return errors.New("could not setup network: in ready state")
		return nil
	}

	logger.Info("starting")

	go n.startDiscovery(logger)

	async.Interval(n.ctx, connManagerGCInterval, n.peersBalancing(logger))
	// don't report metrics in tests
	if n.cfg.Metrics != nil {
		async.Interval(n.ctx, peersReportingInterval, n.reportAllPeers(logger))

		async.Interval(n.ctx, peerIdentitiesReportingInterval, n.reportPeerIdentities(logger))

		async.Interval(n.ctx, topicsReportingInterval, n.reportTopics(logger))
	}

	if err := n.subscribeToSubnets(logger); err != nil {
		return err
	}

	return nil
}

func (n *p2pNetwork) peersBalancing(logger *zap.Logger) func() {
	return func() {
		allPeers := n.host.Network().Peers()
		currentCount := len(allPeers)
		if currentCount < n.cfg.MaxPeers {
			_ = n.idx.GetSubnetsStats() // trigger metrics update
			return
		}
		ctx, cancel := context.WithTimeout(n.ctx, connManagerGCTimeout)
		defer cancel()

		connMgr := peers.NewConnManager(logger, n.libConnManager, n.idx)
		mySubnets := records.Subnets(n.subnets).Clone()
		connMgr.TagBestPeers(logger, n.cfg.MaxPeers-1, mySubnets, allPeers, n.cfg.TopicMaxPeers)
		connMgr.TrimPeers(ctx, logger, n.host.Network())
	}
}

// startDiscovery starts the required services
// it will try to bootstrap discovery service, and inject a connect function.
// the connect function checks if we can connect to the given peer and if so passing it to the backoff connector.
func (n *p2pNetwork) startDiscovery(logger *zap.Logger) {
	discoveredPeers := make(chan peer.AddrInfo, connectorQueueSize)
	go func() {
		ctx, cancel := context.WithCancel(n.ctx)
		defer cancel()
		n.backoffConnector.Connect(ctx, discoveredPeers)
	}()
	err := tasks.Retry(func() error {
		return n.disc.Bootstrap(logger, func(e discovery.PeerEvent) {
			if !n.idx.CanConnect(e.AddrInfo.ID) {
				return
			}
			select {
			case discoveredPeers <- e.AddrInfo:
			default:
				logger.Warn("connector queue is full, skipping new peer", fields.PeerID(e.AddrInfo.ID))
			}
		})
	}, 3)
	if err != nil {
		logger.Panic("could not setup discovery", zap.Error(err))
	}
}

func (n *p2pNetwork) isReady() bool {
	return atomic.LoadInt32(&n.state) == stateReady
}

// UpdateSubnets will update the registered subnets according to active validators
// NOTE: it won't subscribe to the subnets (use subscribeToSubnets for that)
func (n *p2pNetwork) UpdateSubnets(logger *zap.Logger) {
	// TODO: this is a temporary fix to update subnets when validators are added/removed,
	// there is a pending PR to replace this: https://github.com/ssvlabs/ssv/pull/990
	logger = logger.Named(logging.NameP2PNetwork)
	ticker := time.NewTicker(time.Second)
	registeredSubnets := make([]byte, commons.Subnets())
	defer ticker.Stop()

	// Run immediately and then every second.
	for ; true; <-ticker.C {
		start := time.Now()

		// TODO (Alan): Update Metadata.NetworkID to the new domain type after the fork (see Metadata.Subnets below for how to update).

		// Compute the new subnets according to the active validators.
		newSubnets := make([]byte, commons.Subnets())
		copy(newSubnets, n.subnets)
		n.activeValidators.Range(func(pkHex string, status validatorStatus) bool {
			// TODO: alan - fork support
			pbBytes, err := hex.DecodeString(pkHex)
			if err != nil {
				return false
			}
			// TODO: alan - optimization to get active committees only
			share := n.nodeStorage.Shares().Get(nil, pbBytes)
			if share == nil {
				return false
			}
			cid := share.CommitteeID()
			subnet := commons.CommitteeSubnet(cid)
			newSubnets[subnet] = byte(1)
			return true
		})
		n.subnets = newSubnets

		// Compute the not yet registered subnets.
		unregisteredSubnets := make([]int, 0)
		for subnet, active := range newSubnets {
			if active == byte(1) && registeredSubnets[subnet] == byte(0) {
				unregisteredSubnets = append(unregisteredSubnets, subnet)
			}
		}
		registeredSubnets = newSubnets

		if len(unregisteredSubnets) == 0 {
			continue
		}

		self := n.idx.Self()
		self.Metadata.Subnets = records.Subnets(n.subnets).String()
		n.idx.UpdateSelfRecord(self)

		err := n.disc.RegisterSubnets(logger.Named(logging.NameDiscoveryService), unregisteredSubnets...)
		if err != nil {
			logger.Warn("could not register subnets", zap.Error(err))
			continue
		}
		allSubs, _ := records.Subnets{}.FromString(records.AllSubnets)
		subnetsList := records.SharedSubnets(allSubs, n.subnets, 0)
		logger.Debug("updated subnets",
			zap.Any("added", unregisteredSubnets),
			zap.Any("subnets", subnetsList),
			zap.Int("total_subnets", len(subnetsList)),
			zap.Duration("took", time.Since(start)),
		)
	}
}

// UpdateScoreParams updates the scoring parameters once per epoch through the call of n.topicsCtrl.UpdateScoreParams
func (n *p2pNetwork) UpdateScoreParams(logger *zap.Logger) {
	// TODO: this is a temporary solution to update the score parameters periodically.
	// But, we should use an appropriate trigger for the UpdateScoreParams function that should be
	// called once a validator is added or removed from the network

	logger = logger.Named(logging.NameP2PNetwork)

	// Create ticker
	oneEpochDuration := n.cfg.Network.Beacon.SlotDurationSec() * time.Duration(n.cfg.Network.Beacon.SlotsPerEpoch())
	ticker := time.NewTicker(oneEpochDuration)
	defer ticker.Stop()

	// Run immediately and then once every epoch
	for ; true; <-ticker.C {
		err := n.topicsCtrl.UpdateScoreParams(logger)
		if err != nil {
			logger.Debug("score parameters update failed", zap.Error(err))
		} else {
			logger.Debug("updated score parameters successfully")
		}
	}
}

// getMaxPeers returns max peers of the given topic.
func (n *p2pNetwork) getMaxPeers(topic string) int {
	if len(topic) == 0 {
		return n.cfg.MaxPeers
	}
	return n.cfg.TopicMaxPeers
}

// UpdateDomainAtFork updates Domain Type at ENR node record after fork epoch.
func (n *p2pNetwork) UpdateDomainType(logger *zap.Logger, domain spectypes.DomainType) error {
	if err := n.disc.UpdateDomainType(logger, domain); err != nil {
		logger.Error("could not update domain type", zap.Error(err))
		return err
	}
	logger.Debug("updated and published ENR with domain type", fields.Domain(domain))
	return nil
}
