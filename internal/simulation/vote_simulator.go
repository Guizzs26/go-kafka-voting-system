package simulation

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
)

type Simulator struct {
	eventPublisher event.VotePublisher
	concurrency    int
	totalVotes     int
}

func New(ep event.VotePublisher, c, tv int) *Simulator {
	return &Simulator{eventPublisher: ep,
		concurrency: c,
		totalVotes:  tv,
	}
}

func (s *Simulator) Run(ctx context.Context) error {
	log.Printf("Starting stress test: %d votes with %d parallel workers...", s.totalVotes, s.concurrency)

	var wg sync.WaitGroup
	jobs := make(chan struct{}, s.totalVotes)

	for i := 0; i < s.concurrency; i++ {
		wg.Add(1)
		go s.worker(ctx, i+1, &wg, jobs)
	}

	log.Println("Generating vote load...")
	for i := 0; i < s.totalVotes; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	wg.Wait()
	log.Println("Stress test completed. All votes have been published")

	return nil
}

func (s *Simulator) worker(ctx context.Context, id int, wg *sync.WaitGroup, jobs <-chan struct{}) {
	defer wg.Done()
	pollIDs := []string{"poll1", "poll2", "poll3"}
	log.Printf("Worker %d initialized", id)

	for range jobs {
		pollID := pollIDs[rand.Intn(len(pollIDs))]
		userID := fmt.Sprintf("user-%d", rand.Intn(10000))

		vote := model.Vote{
			PollID:    pollID,
			UserID:    userID,
			OptionID:  fmt.Sprintf("option-%d", rand.Intn(3)+1),
			Timestamp: time.Now(),
		}

		publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		if err := s.eventPublisher.PublishMessage(publishCtx, vote, vote.PollID); err != nil {
			log.Printf("[Worker %d] Failed to publish vote: %v", id, err)
		}
		cancel()
	}
	log.Printf("Worker %d finished", id)
}
