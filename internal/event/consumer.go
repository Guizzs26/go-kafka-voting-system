package event

import (
	"context"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
)

type VoteConsumer interface {
	ReadMessage(ctx context.Context) (model.Vote, error)
	Close() error
}
