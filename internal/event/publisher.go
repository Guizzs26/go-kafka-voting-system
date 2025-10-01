package event

import (
	"context"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
)

type VotePublisher interface {
	Publish(ctx context.Context, vote model.Vote, key string) error
	Close() error
}
