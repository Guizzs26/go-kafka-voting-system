package store

import (
	"context"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
)

type VoteStore interface {
	RegisterVote(ctx context.Context, vote model.Vote) (bool, error)
	GetResults(ctx context.Context, pollID string) (map[string]int, error)
	Close() error
}
