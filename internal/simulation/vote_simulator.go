package simulation

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
)

type Simulator struct {
	eventPublisher event.VotePublisher
}

func New(ep event.VotePublisher) *Simulator {
	return &Simulator{eventPublisher: ep}
}

func (s *Simulator) Run(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	const fraudFrequency = 5
	var fraudCounter int
	var lastLegitUserID string
	var lastLegitPollID string

	pollIDs := []string{"poll1", "poll2", "poll3"}
	for {
		select {
		case <-ctx.Done():
			log.Println("Simulator received shutdown signal")
			return nil

		case <-ticker.C:
			var userID, pollID string
			fraudCounter++
			if fraudCounter >= fraudFrequency && lastLegitUserID != "" {
				log.Println("generating a duplicate vote on purpose")
				userID = lastLegitUserID
				pollID = lastLegitPollID
				fraudCounter = 0
			} else {
				pollID = pollIDs[rand.Intn(len(pollIDs))]
				userID = fmt.Sprintf("user-%d", rand.Intn(1000))

				lastLegitUserID = userID
				lastLegitPollID = pollID
			}

			v := model.Vote{
				PollID:    pollID,
				UserID:    userID,
				OptionID:  fmt.Sprintf("option-%d", rand.Intn(3)+1),
				Timestamp: time.Now(),
			}

			publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			log.Printf("Generating vote: PollID=%s, UserID=%s", v.PollID, v.UserID)
			if err := s.eventPublisher.PublishMessage(publishCtx, v, pollID); err != nil {
				log.Printf("Failed to publish vote: %v", err)
			}
			cancel()
		}
	}
}
