package ibft

import (
	"bytes"
	"encoding/hex"

	"github.com/bloxapp/ssv/ibft/proto"

	"go.uber.org/zap"
)

func (i *Instance) commitMsgPipeline() Pipeline {
	return []PipelineFunc{
		MsgTypeCheck(proto.RoundState_Commit),
		i.ValidateLambdas(),
		i.ValidateRound(),
		i.AuthMsg(),
		i.validateCommitMsg(),
		i.uponCommitMsg(),
	}
}

func (i *Instance) validateCommitMsg() PipelineFunc {
	return func(signedMessage *proto.SignedMessage) error {
		// TODO - should we test prepared round as well?

		if err := i.consensus.Validate(signedMessage.Message.Value); err != nil {
			return err
		}

		return nil
	}
}

// TODO - passing round can be problematic if the node goes down, it might not know which round it is now.
func (i *Instance) commitQuorum(round uint64, inputValue []byte) (quorum bool, t int, n int) {
	// TODO - do we need to validate round?
	cnt := 0
	msgs := i.commitMessages.ReadOnlyMessagesByRound(round)
	for _, v := range msgs {
		if bytes.Equal(inputValue, v.Message.Value) {
			cnt += 1
		}
	}
	quorum = cnt*3 >= i.params.CommitteeSize()*2
	return quorum, cnt, i.params.CommitteeSize()
}

func (i *Instance) existingCommitMsg(signedMessage *proto.SignedMessage) bool {
	msgs := i.commitMessages.ReadOnlyMessagesByRound(signedMessage.Message.Round)
	for _, id := range signedMessage.SignerIds {
		if _, found := msgs[id]; found {
			return true
		}
	}

	return false
}

/**
upon receiving a quorum Qcommit of valid ⟨COMMIT, λi, round, value⟩ messages do:
	set timer i to stopped
	Decide(λi , value, Qcommit)
*/
func (i *Instance) uponCommitMsg() PipelineFunc {
	// TODO - concurrency lock?
	return func(signedMessage *proto.SignedMessage) error {
		// Only 1 prepare per peer per round is valid
		if i.existingCommitMsg(signedMessage) {
			return nil
		}

		// add to prepare messages
		i.commitMessages.AddMessage(signedMessage)
		i.logger.Info("received valid commit message for round",
			zap.String("sender_ibft_id", signedMessage.SignersIdString()),
			zap.Uint64("round", signedMessage.Message.Round))

		// check if quorum achieved, act upon it.
		if i.State.Stage == proto.RoundState_Decided {
			return nil // no reason to commit again
		}
		quorum, t, n := i.commitQuorum(i.State.PreparedRound, i.State.PreparedValue)
		if quorum { // if already decidedChan no need to do it again
			i.logger.Info("decided iBFT instance",
				zap.String("Lambda", hex.EncodeToString(i.State.Lambda)), zap.Uint64("round", i.State.Round),
				zap.Int("got_votes", t), zap.Int("total_votes", n))

			// mark instance decided
			i.SetStage(proto.RoundState_Decided)
			i.stopRoundChangeTimer()
		}
		return nil
	}
}
