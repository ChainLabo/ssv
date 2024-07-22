package topics

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	eth2apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	eth2types "github.com/wealdtech/go-eth2-types/v2"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	specqbft "github.com/ssvlabs/ssv-spec/qbft"
	spectypes "github.com/ssvlabs/ssv-spec/types"
	spectestingutils "github.com/ssvlabs/ssv-spec/types/testingutils"
	"github.com/ssvlabs/ssv/logging"
	"github.com/ssvlabs/ssv/message/signatureverifier"
	"github.com/ssvlabs/ssv/message/validation"
	"github.com/ssvlabs/ssv/monitoring/metricsreporter"
	"github.com/ssvlabs/ssv/network/commons"
	"github.com/ssvlabs/ssv/network/discovery"
	"github.com/ssvlabs/ssv/networkconfig"
	"github.com/ssvlabs/ssv/operator/duties/dutystore"
	"github.com/ssvlabs/ssv/operator/storage"
	beaconprotocol "github.com/ssvlabs/ssv/protocol/v2/blockchain/beacon"
	ssvtypes "github.com/ssvlabs/ssv/protocol/v2/types"
	"github.com/ssvlabs/ssv/registry/storage/mocks"
	"github.com/ssvlabs/ssv/storage/basedb"
	"github.com/ssvlabs/ssv/storage/kv"
)

func TestTopicManager(t *testing.T) {
	logger := logging.TestLogger(t)

	// TODO: rework this test to use message validation
	t.Run("happy flow", func(t *testing.T) {
		nPeers := 4

		pks := []string{"b768cdc2b2e0a859052bf04d1cd66383c96d95096a5287d08151494ce709556ba39c1300fbb902a0e2ebb7c31dc4e400",
			"824b9024767a01b56790a72afb5f18bb0f97d5bddb946a7bd8dd35cc607c35a4d76be21f24f484d0d478b99dc63ed170",
			"9340b7b80983a412bbb42cad6f992e06983d53deb41166ed5978dcbfa3761f347b237ad446d7cb4a4d0a5cca78c2ce8a",
			"a5abb232568fc869765da01688387738153f3ad6cc4e635ab282c5d5cfce2bba2351f03367103090804c5243dc8e229b",
			"a1169bd8407279d9e56b8cefafa37449afd6751f94d1da6bc8145b96d7ad2940184d506971291cd55ae152f9fc65b146",
			"80ff2cfb8fd80ceafbb3c331f271a9f9ce0ed3e360087e314d0a8775e86fa7cd19c999b821372ab6419cde376e032ff6",
			"a01909aac48337bab37c0dba395fb7495b600a53c58059a251d00b4160b9da74c62f9c4e9671125c59932e7bb864fd3d",
			"a4fc8c859ed5c10d7a1ff9fb111b76df3f2e0a6cbe7d0c58d3c98973c0ff160978bc9754a964b24929fff486ebccb629"}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		validator := &DummyMessageValidator{}

		peers := newPeers(ctx, logger, t, nPeers, validator, true, nil)
		baseTest(t, ctx, logger, peers, pks, 1, 2)
	})

	t.Run("banning peer", func(t *testing.T) {
		t.Skip() // TODO: finish the test

		pks := []string{
			"b768cdc2b2e0a859052bf04d1cd66383c96d95096a5287d08151494ce709556ba39c1300fbb902a0e2ebb7c31dc4e400",
			"824b9024767a01b56790a72afb5f18bb0f97d5bddb946a7bd8dd35cc607c35a4d76be21f24f484d0d478b99dc63ed170",
			"9340b7b80983a412bbb42cad6f992e06983d53deb41166ed5978dcbfa3761f347b237ad446d7cb4a4d0a5cca78c2ce8a",
			"a5abb232568fc869765da01688387738153f3ad6cc4e635ab282c5d5cfce2bba2351f03367103090804c5243dc8e229b",
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctrl := gomock.NewController(t)
		logger := zaptest.NewLogger(t)
		db, err := kv.NewInMemory(logger, basedb.Options{})
		require.NoError(t, err)
		ns, err := storage.NewNodeStorage(logger, db)
		require.NoError(t, err)
		dutyStore := dutystore.New()
		ks := spectestingutils.Testing4SharesSet()
		shares := generateShares(t, ks, ns, networkconfig.TestNetwork)
		validatorStore := mocks.NewMockValidatorStore(ctrl)
		validatorStore.EXPECT().Validator(gomock.Any()).DoAndReturn(func(pubKey []byte) *ssvtypes.SSVShare {
			for _, share := range []*ssvtypes.SSVShare{
				shares.active,
				shares.liquidated,
				shares.inactive,
				shares.nonUpdatedMetadata,
				shares.nonUpdatedMetadataNextEpoch,
				shares.noMetadata,
			} {
				if bytes.Equal(share.ValidatorPubKey[:], pubKey) {
					return share
				}
			}
			return nil
		}).AnyTimes()
		signatureVerifier := signatureverifier.NewMockSignatureVerifier(ctrl)
		signatureVerifier.EXPECT().VerifySignature(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		validator := validation.New(networkconfig.TestNetwork, validatorStore, dutyStore, signatureVerifier)

		scoreMap := map[peer.ID]*pubsub.PeerScoreSnapshot{}
		var scoreMapMu sync.Mutex

		scoreInspector := func(m map[peer.ID]*pubsub.PeerScoreSnapshot) {
			b, _ := json.Marshal(m)
			t.Logf("peer scores: %v", string(b))

			scoreMapMu.Lock()
			defer scoreMapMu.Unlock()

			scoreMap = m
		}

		const nPeers = 4
		peers := newPeers(ctx, logger, t, nPeers, validator, true, scoreInspector)
		banningTest(t, ctx, logger, peers, pks, scoreMap, &scoreMapMu)
	})
}

func baseTest(t *testing.T, ctx context.Context, logger *zap.Logger, peers []*P, pks []string, minMsgCount, maxMsgCount int) {
	nValidators := len(pks)
	// nPeers := len(peers)

	t.Log("subscribing to topics")
	// listen to topics
	for _, pk := range pks {
		for _, p := range peers {
			require.NoError(t, p.tm.Subscribe(logger, validatorTopic(pk)))
			// simulate concurrency, by trying to subscribe multiple times
			go func(tm Controller, pk string) {
				require.NoError(t, tm.Subscribe(logger, validatorTopic(pk)))
			}(p.tm, pk)
			go func(tm Controller, pk string) {
				<-time.After(100 * time.Millisecond)
				require.NoError(t, tm.Subscribe(logger, validatorTopic(pk)))
			}(p.tm, pk)
		}
	}

	// wait for the peers to join topics
	<-time.After(3 * time.Second)

	t.Log("broadcasting messages")

	var wg sync.WaitGroup
	// publish some messages
	for i := 0; i < nValidators; i++ {
		for j, p := range peers {
			wg.Add(1)
			go func(p *P, pk string, pi int) {
				defer wg.Done()
				msg, err := dummyMsg(pk, pi%4, false)
				require.NoError(t, err)
				raw, err := msg.Encode()
				require.NoError(t, err)
				require.NoError(t, p.tm.Broadcast(validatorTopic(pk), raw, time.Second*10))
				<-time.After(time.Second * 5)
			}(p, pks[i], j)
		}
	}
	wg.Wait()

	// let the messages propagate
	wg.Add(1)
	go func() {
		defer wg.Done()
		// check number of peers and messages
		for i := 0; i < nValidators; i++ {
			wg.Add(1)
			go func(pk string) {
				ctxReadMessages, cancel := context.WithTimeout(ctx, time.Second*5)
				defer cancel()
				defer wg.Done()
				for _, p := range peers {
					// wait for messages
					for ctxReadMessages.Err() == nil && p.getCount(commons.GetTopicFullName(validatorTopic(pk))) < minMsgCount {
						time.Sleep(time.Millisecond * 100)
					}
					require.NoError(t, ctxReadMessages.Err())
					c := p.getCount(commons.GetTopicFullName(validatorTopic(pk)))
					require.GreaterOrEqual(t, c, minMsgCount)
					// require.LessOrEqual(t, c, maxMsgCount)
				}
			}(pks[i])
		}
	}()
	wg.Wait()

	t.Log("unsubscribing")
	// unsubscribing multiple times for each topic
	for i := 0; i < nValidators; i++ {
		for _, p := range peers {
			wg.Add(1)
			go func(p *P, pk string) {
				defer wg.Done()
				require.NoError(t, p.tm.Unsubscribe(logger, validatorTopic(pk), false))
				go func(p *P) {
					<-time.After(time.Millisecond)
					require.NoError(t, p.tm.Unsubscribe(logger, validatorTopic(pk), false))
				}(p)
				wg.Add(1)
				go func(p *P) {
					defer wg.Done()
					<-time.After(time.Millisecond * 50)
					require.NoError(t, p.tm.Unsubscribe(logger, validatorTopic(pk), false))
				}(p)
			}(p, pks[i])
		}
	}
	wg.Wait()
}

func banningTest(t *testing.T, ctx context.Context, logger *zap.Logger, peers []*P, pks []string, scoreMap map[peer.ID]*pubsub.PeerScoreSnapshot, scoreMapMu *sync.Mutex) {
	t.Log("subscribing to topics")

	for _, pk := range pks {
		for _, p := range peers {
			require.NoError(t, p.tm.Subscribe(logger, validatorTopic(pk)))
		}
	}

	// wait for the peers to join topics
	<-time.After(3 * time.Second)

	t.Log("checking initial scores")
	for _, pk := range pks {
		for _, p := range peers {
			peerList, err := p.tm.Peers(pk)
			require.NoError(t, err)

			for _, pid := range peerList {
				scoreMapMu.Lock()
				v, ok := scoreMap[pid]
				scoreMapMu.Unlock()

				require.True(t, ok)
				require.Equal(t, 0, v.Score)
			}
		}
	}

	t.Log("broadcasting messages")

	const invalidMessagesCount = 10

	// TODO: get current default score, send an invalid rejected message, check the score; then run 10 of them and check the score; then check valid message

	invalidMessages, err := msgSequence(pks[0], invalidMessagesCount, len(pks), true)
	require.NoError(t, err)

	var wg sync.WaitGroup
	// publish some messages
	for i, msg := range invalidMessages {
		wg.Add(1)
		go func(p *P, pk string, msg *spectypes.SignedSSVMessage) {
			defer wg.Done()

			raw, err := msg.Encode()
			require.NoError(t, err)

			require.NoError(t, p.tm.Broadcast(validatorTopic(pk), raw, time.Second*10))

			<-time.After(time.Second * 5)
		}(peers[0], pks[i%len(pks)], msg)
	}
	wg.Wait()

	<-time.After(5 * time.Second)

	t.Log("checking final scores")
	for _, pk := range pks {
		for _, p := range peers {
			peerList, err := p.tm.Peers(pk)
			require.NoError(t, err)

			for _, pid := range peerList {
				scoreMapMu.Lock()
				v, ok := scoreMap[pid]
				scoreMapMu.Unlock()

				require.True(t, ok)
				require.Equal(t, 0, v.Score) // TODO: score should change
			}
		}
	}

	//t.Log("unsubscribing")
	//// unsubscribing multiple times for each topic
	//wg.Add(1)
	//go func(p *P, pk string) {
	//	defer wg.Done()
	//	require.NoError(t, p.tm.Unsubscribe(logger, validatorTopic(pk), false))
	//	go func(p *P) {
	//		<-time.After(time.Millisecond)
	//		require.NoError(t, p.tm.Unsubscribe(logger, validatorTopic(pk), false))
	//	}(p)
	//	wg.Add(1)
	//	go func(p *P) {
	//		defer wg.Done()
	//		<-time.After(time.Millisecond * 50)
	//		require.NoError(t, p.tm.Unsubscribe(logger, validatorTopic(pk), false))
	//	}(p)
	//}(peer, pk)
	//
	//wg.Wait()
}

func validatorTopic(pkhex string) string {
	pk, err := hex.DecodeString(pkhex)
	if err != nil {
		return "invalid"
	}
	return commons.ValidatorTopicID(pk)[0]
}

type P struct {
	host host.Host
	ps   *pubsub.PubSub
	tm   *topicsCtrl

	connsCount uint64

	msgsLock sync.Locker
	msgs     map[string][]*pubsub.Message
}

func (p *P) getCount(t string) int {
	p.msgsLock.Lock()
	defer p.msgsLock.Unlock()

	msgs, ok := p.msgs[t]
	if !ok {
		return 0
	}
	return len(msgs)
}

func (p *P) saveMsg(t string, msg *pubsub.Message) {
	p.msgsLock.Lock()
	defer p.msgsLock.Unlock()

	msgs, ok := p.msgs[t]
	if !ok {
		msgs = make([]*pubsub.Message, 0)
	}
	msgs = append(msgs, msg)
	p.msgs[t] = msgs
}

// TODO: use p2p/testing
func newPeers(ctx context.Context, logger *zap.Logger, t *testing.T, n int, msgValidator validation.MessageValidator, msgID bool, scoreInspector pubsub.ExtendedPeerScoreInspectFn) []*P {
	peers := make([]*P, n)
	for i := 0; i < n; i++ {
		peers[i] = newPeer(ctx, logger, t, msgValidator, msgID, scoreInspector)
	}
	t.Logf("%d peers were created", n)
	th := uint64(n/2) + uint64(n/4)
	for ctx.Err() == nil {
		done := 0
		for _, p := range peers {
			if atomic.LoadUint64(&p.connsCount) >= th {
				done++
			}
		}
		if done == len(peers) {
			break
		}
	}
	t.Log("peers are connected")
	return peers
}

func newPeer(ctx context.Context, logger *zap.Logger, t *testing.T, msgValidator validation.MessageValidator, msgID bool, scoreInspector pubsub.ExtendedPeerScoreInspectFn) *P {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)
	ds, err := discovery.NewLocalDiscovery(ctx, logger, h)
	require.NoError(t, err)

	var p *P
	var midHandler MsgIDHandler
	if msgID {
		midHandler = NewMsgIDHandler(ctx, 2*time.Minute)
		go midHandler.Start()
	}
	cfg := &PubSubConfig{
		Host:         h,
		TraceLog:     false,
		MsgIDHandler: midHandler,
		MsgHandler: func(_ context.Context, topic string, msg *pubsub.Message) error {
			p.saveMsg(topic, msg)
			return nil
		},
		Scoring: &ScoringConfig{
			IPWhilelist:        nil,
			IPColocationWeight: 0,
			OneEpochDuration:   time.Minute,
		},
		MsgValidator:           msgValidator,
		ScoreInspector:         scoreInspector,
		ScoreInspectorInterval: 100 * time.Millisecond,
		// TODO: add mock for peers.ScoreIndex
	}

	ps, tm, err := NewPubSub(ctx, logger, cfg, metricsreporter.NewNop())
	require.NoError(t, err)

	p = &P{
		host:     h,
		ps:       ps,
		tm:       tm.(*topicsCtrl),
		msgs:     make(map[string][]*pubsub.Message),
		msgsLock: &sync.Mutex{},
	}
	h.Network().Notify(&libp2pnetwork.NotifyBundle{
		ConnectedF: func(network libp2pnetwork.Network, conn libp2pnetwork.Conn) {
			atomic.AddUint64(&p.connsCount, 1)
		},
	})
	require.NoError(t, ds.Bootstrap(logger, func(e discovery.PeerEvent) {
		_ = h.Connect(ctx, e.AddrInfo)
	}))

	return p
}

func msgSequence(pkHex string, n, committeeSize int, malformed bool) ([]*spectypes.SignedSSVMessage, error) {
	var messages []*spectypes.SignedSSVMessage

	for i := 0; i < n; i++ {
		height := i * committeeSize
		msg, err := dummyMsg(pkHex, height, malformed)
		if err != nil {
			return nil, err
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func dummyMsg(pkHex string, height int, malformed bool) (*spectypes.SignedSSVMessage, error) {
	pk, err := hex.DecodeString(pkHex)
	if err != nil {
		return nil, err
	}

	id := spectypes.NewMsgID(networkconfig.TestNetwork.DomainType(), pk, spectypes.RunnerRole(spectypes.BNRoleAttester))
	signature, err := base64.StdEncoding.DecodeString("sVV0fsvqQlqliKv/ussGIatxpe8LDWhc9uoaM5WpjbiYvvxUr1eCpz0ja7UT1PGNDdmoGi6xbMC1g/ozhAt4uCdpy0Xdfqbv2hMf2iRL5ZPKOSmMifHbd8yg4PeeceyN")
	if err != nil {
		return nil, err
	}
	msg := &specqbft.Message{
		MsgType:    specqbft.RoundChangeMsgType,
		Height:     specqbft.Height(height),
		Round:      2,
		Identifier: id[:],
		Root:       [32]byte{},
	}
	encMsg, err := msg.Encode()
	ssvMessage := &spectypes.SSVMessage{
		MsgType: spectypes.SSVConsensusMsgType,
		MsgID:   id,
		Data:    encMsg,
	}
	if malformed {
		ssvMessage.MsgType = math.MaxUint64
	}

	signedMessage := &spectypes.SignedSSVMessage{
		Signatures:  [][]byte{signature},
		OperatorIDs: []spectypes.OperatorID{1, 3, 4},
		SSVMessage:  ssvMessage,
		FullData:    nil,
	}

	return signedMessage, nil
}

type DummyMessageValidator struct {
}

func (m *DummyMessageValidator) ValidatorForTopic(topic string) func(ctx context.Context, p peer.ID, pmsg *pubsub.Message) pubsub.ValidationResult {
	return func(ctx context.Context, p peer.ID, pmsg *pubsub.Message) pubsub.ValidationResult {
		return pubsub.ValidationAccept
	}
}

func (m *DummyMessageValidator) ValidatePubsubMessage(ctx context.Context, p peer.ID, pmsg *pubsub.Message) pubsub.ValidationResult {
	return pubsub.ValidationAccept
}

func (m *DummyMessageValidator) Validate(ctx context.Context, p peer.ID, pmsg *pubsub.Message) pubsub.ValidationResult {
	return 0
}

type shareSet struct {
	active                      *ssvtypes.SSVShare
	liquidated                  *ssvtypes.SSVShare
	inactive                    *ssvtypes.SSVShare
	nonUpdatedMetadata          *ssvtypes.SSVShare
	nonUpdatedMetadataNextEpoch *ssvtypes.SSVShare
	noMetadata                  *ssvtypes.SSVShare
}

func generateShares(t *testing.T, ks *spectestingutils.TestKeySet, ns storage.Storage, netCfg networkconfig.NetworkConfig) shareSet {
	activeShare := &ssvtypes.SSVShare{
		Share: *spectestingutils.TestingShare(ks, spectestingutils.TestingValidatorIndex),
		Metadata: ssvtypes.Metadata{
			BeaconMetadata: &beaconprotocol.ValidatorMetadata{
				Status: eth2apiv1.ValidatorStateActiveOngoing,
				Index:  spectestingutils.TestingShare(ks, spectestingutils.TestingValidatorIndex).ValidatorIndex,
			},
			Liquidated: false,
		},
	}

	require.NoError(t, ns.Shares().Save(nil, activeShare))

	liquidatedShare := &ssvtypes.SSVShare{
		Share: *spectestingutils.TestingShare(ks, spectestingutils.TestingValidatorIndex),
		Metadata: ssvtypes.Metadata{
			BeaconMetadata: &beaconprotocol.ValidatorMetadata{
				Status: eth2apiv1.ValidatorStateActiveOngoing,
				Index:  spectestingutils.TestingShare(ks, spectestingutils.TestingValidatorIndex).ValidatorIndex,
			},
			Liquidated: true,
		},
	}

	liquidatedSK, err := eth2types.GenerateBLSPrivateKey()
	require.NoError(t, err)

	copy(liquidatedShare.ValidatorPubKey[:], liquidatedSK.PublicKey().Marshal())
	require.NoError(t, ns.Shares().Save(nil, liquidatedShare))

	inactiveShare := &ssvtypes.SSVShare{
		Share: *spectestingutils.TestingShare(ks, spectestingutils.TestingValidatorIndex),
		Metadata: ssvtypes.Metadata{
			BeaconMetadata: &beaconprotocol.ValidatorMetadata{
				Status: eth2apiv1.ValidatorStateUnknown,
			},
			Liquidated: false,
		},
	}

	inactiveSK, err := eth2types.GenerateBLSPrivateKey()
	require.NoError(t, err)

	copy(inactiveShare.ValidatorPubKey[:], inactiveSK.PublicKey().Marshal())
	require.NoError(t, ns.Shares().Save(nil, inactiveShare))

	slot := netCfg.Beacon.EstimatedCurrentSlot()
	epoch := netCfg.Beacon.EstimatedEpochAtSlot(slot)

	nonUpdatedMetadataShare := &ssvtypes.SSVShare{
		Share: *spectestingutils.TestingShare(ks, spectestingutils.TestingValidatorIndex),
		Metadata: ssvtypes.Metadata{
			BeaconMetadata: &beaconprotocol.ValidatorMetadata{
				Status:          eth2apiv1.ValidatorStatePendingQueued,
				ActivationEpoch: epoch,
			},
			Liquidated: false,
		},
	}

	nonUpdatedMetadataSK, err := eth2types.GenerateBLSPrivateKey()
	require.NoError(t, err)

	copy(nonUpdatedMetadataShare.ValidatorPubKey[:], nonUpdatedMetadataSK.PublicKey().Marshal())
	require.NoError(t, ns.Shares().Save(nil, nonUpdatedMetadataShare))

	nonUpdatedMetadataNextEpochShare := &ssvtypes.SSVShare{
		Share: *spectestingutils.TestingShare(ks, spectestingutils.TestingValidatorIndex),
		Metadata: ssvtypes.Metadata{
			BeaconMetadata: &beaconprotocol.ValidatorMetadata{
				Status:          eth2apiv1.ValidatorStatePendingQueued,
				ActivationEpoch: epoch + 1,
			},
			Liquidated: false,
		},
	}

	nonUpdatedMetadataNextEpochSK, err := eth2types.GenerateBLSPrivateKey()
	require.NoError(t, err)

	copy(nonUpdatedMetadataNextEpochShare.ValidatorPubKey[:], nonUpdatedMetadataNextEpochSK.PublicKey().Marshal())
	require.NoError(t, ns.Shares().Save(nil, nonUpdatedMetadataNextEpochShare))

	noMetadataShare := &ssvtypes.SSVShare{
		Share: *spectestingutils.TestingShare(ks, spectestingutils.TestingValidatorIndex),
		Metadata: ssvtypes.Metadata{
			BeaconMetadata: nil,
			Liquidated:     false,
		},
	}

	noMetadataShareSK, err := eth2types.GenerateBLSPrivateKey()
	require.NoError(t, err)

	copy(noMetadataShare.ValidatorPubKey[:], noMetadataShareSK.PublicKey().Marshal())
	require.NoError(t, ns.Shares().Save(nil, noMetadataShare))

	return shareSet{
		active:                      activeShare,
		liquidated:                  liquidatedShare,
		inactive:                    inactiveShare,
		nonUpdatedMetadata:          nonUpdatedMetadataShare,
		nonUpdatedMetadataNextEpoch: nonUpdatedMetadataNextEpochShare,
		noMetadata:                  noMetadataShare,
	}
}
