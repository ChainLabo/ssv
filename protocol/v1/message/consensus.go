package message

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/pkg/errors"
)

// ConsensusMessageType is the type of consensus messages
type ConsensusMessageType int

const (
	// ProposalMsgType is the type used for proposal messages
	ProposalMsgType ConsensusMessageType = iota
	// PrepareMsgType is the type used for prepare messages
	PrepareMsgType
	// CommitMsgType is the type used for commit messages
	CommitMsgType
	// RoundChangeMsgType is the type used for change round messages
	RoundChangeMsgType
	// DecidedMsgType is the type used for decided messages
	DecidedMsgType
)

// ProposalData is the structure used for propose messages
type ProposalData struct {
	Data                     []byte
	RoundChangeJustification []*SignedMessage
	PrepareJustification     []*SignedMessage
}

// Encode returns a msg encoded bytes or error
func (d *ProposalData) Encode() ([]byte, error) {
	return json.Marshal(d)
}

// Decode returns error if decoding failed
func (d *ProposalData) Decode(data []byte) error {
	return json.Unmarshal(data, &d)
}

// PrepareData is the structure used for prepare messages
type PrepareData struct {
	Data []byte
}

// Encode returns a msg encoded bytes or error
func (d *PrepareData) Encode() ([]byte, error) {
	return json.Marshal(d)
}

// Decode returns error if decoding failed
func (d *PrepareData) Decode(data []byte) error {
	return json.Unmarshal(data, &d)
}

// CommitData is the structure used for commit messages
type CommitData struct {
	Data []byte
}

// Encode returns a msg encoded bytes or error
func (d *CommitData) Encode() ([]byte, error) {
	return json.Marshal(d)
}

// Decode returns error if decoding failed
func (d *CommitData) Decode(data []byte) error {
	return json.Unmarshal(data, &d)
}

// Round is the QBFT round of the message
type Round uint64

// Height is the height of the QBFT instance
type Height int64

// RoundChangeData represents the data that is sent upon change round
type RoundChangeData interface {
	GetPreparedValue() []byte
	GetPreparedRound() Round
	// GetNextProposalData returns NOT nil byte array if the signer is the next round's proposal.
	GetNextProposalData() []byte
	// GetRoundChangeJustification returns signed prepare messages for the last prepared state
	GetRoundChangeJustification() []*SignedMessage
}

// ConsensusMessage is the structure used for consensus messages
type ConsensusMessage struct {
	MsgType    ConsensusMessageType
	Height     Height // QBFT instance Height
	Round      Round  // QBFT round for which the msg is for
	Identifier []byte // instance Identifier this msg belongs to
	Data       []byte
}

// GetProposalData returns proposal specific data
func (msg *ConsensusMessage) GetProposalData() (*ProposalData, error) {
	ret := &ProposalData{}
	if err := ret.Decode(msg.Data); err != nil {
		return nil, errors.Wrap(err, "could not decode proposal data from message")
	}
	return ret, nil
}

// GetPrepareData returns prepare specific data
func (msg *ConsensusMessage) GetPrepareData() (*PrepareData, error) {
	ret := &PrepareData{}
	if err := ret.Decode(msg.Data); err != nil {
		return nil, errors.Wrap(err, "could not decode prepare data from message")
	}
	return ret, nil
}

// GetCommitData returns commit specific data
func (msg *ConsensusMessage) GetCommitData() (*CommitData, error) {
	ret := &CommitData{}
	if err := ret.Decode(msg.Data); err != nil {
		return nil, errors.Wrap(err, "could not decode commit data from message")
	}
	return ret, nil
}

// GetRoundChangeData returns round change specific data
func (msg *ConsensusMessage) GetRoundChangeData() RoundChangeData {
	panic("implement")
}

// Encode returns a msg encoded bytes or error
func (msg *ConsensusMessage) Encode() ([]byte, error) {
	return json.Marshal(msg)
}

// Decode returns error if decoding failed
func (msg *ConsensusMessage) Decode(data []byte) error {
	return json.Unmarshal(data, &msg)
}

// GetRoot returns the root used for signing and verification
func (msg *ConsensusMessage) GetRoot() ([]byte, error) {
	marshaledRoot, err := msg.Encode()
	if err != nil {
		return nil, errors.Wrap(err, "could not encode message")
	}
	ret := sha256.Sum256(marshaledRoot)
	return ret[:], nil
}

// DeepCopy returns a new instance of ConsensusMessage, deep copied
func (msg *ConsensusMessage) DeepCopy() *ConsensusMessage {
	panic("implement")
}

// SignedMessage contains a message and the corresponding signature + signers list
type SignedMessage struct {
	Signature Signature
	Signers   []OperatorID
	Message   *ConsensusMessage // message for which this signature is for
}

// GetSignature returns the message signature
func (signedMsg *SignedMessage) GetSignature() Signature {
	return signedMsg.Signature
}

// GetSigners returns the message signers
func (signedMsg *SignedMessage) GetSigners() []OperatorID {
	return signedMsg.Signers
}

// MatchedSigners returns true if the provided signer ids are equal to GetSignerIds() without order significance
func (signedMsg *SignedMessage) MatchedSigners(ids []OperatorID) bool {
	for _, id := range signedMsg.Signers {
		found := false
		for _, id2 := range ids {
			if id == id2 {
				found = true
			}
		}

		if !found {
			return false
		}
	}
	return true
}

// MutualSigners returns true if signatures have at least 1 mutual signer
func (signedMsg *SignedMessage) MutualSigners(sig MsgSignature) bool {
	for _, id := range signedMsg.Signers {
		for _, id2 := range sig.GetSigners() {
			if id == id2 {
				return true
			}
		}
	}
	return false
}

// Aggregate will aggregate the signed message if possible (unique signers, same digest, valid)
func (signedMsg *SignedMessage) Aggregate(sig MsgSignature) error {
	if signedMsg.MutualSigners(sig) {
		return errors.New("can't aggregate 2 signed messages with mutual signers")
	}

	aggregated, err := signedMsg.Signature.Aggregate(sig.GetSignature())
	if err != nil {
		return errors.Wrap(err, "could not aggregate signatures")
	}
	signedMsg.Signature = aggregated
	signedMsg.Signers = append(signedMsg.Signers, sig.GetSigners()...)

	return nil
}

// Encode returns a msg encoded bytes or error
func (signedMsg *SignedMessage) Encode() ([]byte, error) {
	return json.Marshal(signedMsg)
}

// Decode returns error if decoding failed
func (signedMsg *SignedMessage) Decode(data []byte) error {
	return json.Unmarshal(data, &signedMsg)
}

// GetRoot returns the root used for signing and verification
func (signedMsg *SignedMessage) GetRoot() ([]byte, error) {
	return signedMsg.Message.GetRoot()
}

// DeepCopy returns a new instance of SignedMessage, deep copied
func (signedMsg *SignedMessage) DeepCopy() *SignedMessage {
	ret := &SignedMessage{
		Signers:   make([]OperatorID, len(signedMsg.Signers)),
		Signature: make([]byte, len(signedMsg.Signature)),
	}
	copy(ret.Signers, signedMsg.Signers)
	copy(ret.Signature, signedMsg.Signature)

	ret.Message = &ConsensusMessage{
		MsgType:    signedMsg.Message.MsgType,
		Height:     signedMsg.Message.Height,
		Round:      signedMsg.Message.Round,
		Identifier: make([]byte, len(signedMsg.Message.Identifier)),
		Data:       make([]byte, len(signedMsg.Message.Data)),
	}
	copy(ret.Message.Identifier, signedMsg.Message.Identifier)
	copy(ret.Message.Data, signedMsg.Message.Data)
	return ret
}
