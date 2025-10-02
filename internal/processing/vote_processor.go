package processing

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/metrics"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/store"
)

type VoteProcessor struct {
	consumer  event.VoteConsumer
	publisher event.VotePublisher
	metrics   *metrics.ProcessorMetrics
	store     store.VoteStore

	// we maintain minimal, local state: just the IDs of polls we've already seen
	mu         sync.Mutex
	knownPolls map[string]bool
}

func NewVoteProcessor(
	c event.VoteConsumer,
	p event.VotePublisher,
	m *metrics.ProcessorMetrics,
	s store.VoteStore,
) *VoteProcessor {
	return &VoteProcessor{
		consumer:   c,
		publisher:  p,
		metrics:    m,
		store:      s,
		knownPolls: make(map[string]bool),
	}
}

func (vp *VoteProcessor) Run(ctx context.Context) error {
	rTicker := time.NewTicker(5 * time.Second)
	defer rTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Vote processor receiving signal to stop.")
			return nil

		case <-rTicker.C:
			vp.printResults(ctx)

		default:
			v, err := vp.consumer.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() == context.Canceled {
					continue
				}
				log.Printf("Error reading message: %v", err)
				continue
			}
			vp.processVote(ctx, v)
		}
	}
}

func (vp *VoteProcessor) processVote(ctx context.Context, v model.Vote) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		vp.metrics.ProcessingTime.WithLabelValues(v.PollID).Observe(duration)
	}()

	isNewVote, err := vp.store.RegisterVote(ctx, v)
	if err != nil {
		log.Printf("Error registering vote: %v", err)
		return
	}

	vp.mu.Lock()
	vp.knownPolls[v.PollID] = true
	vp.mu.Unlock()

	if !isNewVote {
		log.Printf("[FRAUD DETECTED] Duplicate vote from UserID: %s to PollID: %s", v.UserID, v.PollID)
		vp.metrics.VotesDuplicate.WithLabelValues(v.PollID).Inc()

		dlqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := vp.publisher.PublishMessage(dlqCtx, v, v.PollID); err != nil {
			log.Printf("[CRITICAL ERROR] Failed to publishing to DLQ: %v", err)
		}
		return // we're done here
	}

	log.Printf("[VALID VOTE] UserID: %s voted for OptionID: %s in PollID: %s", v.UserID, v.OptionID, v.PollID)
	vp.metrics.VotesProcessed.WithLabelValues(v.PollID).Inc()
}

func (vp *VoteProcessor) printResults(ctx context.Context) {
	vp.mu.Lock()
	pollIDs := make([]string, 0, len(vp.knownPolls))
	for id := range vp.knownPolls {
		pollIDs = append(pollIDs, id)
	}
	vp.mu.Unlock()

	log.Println("--- CURRENT SCORE (Redis) ---")
	if len(pollIDs) == 0 {
		log.Println("No polls processed yet")
		return
	}

	for _, poolID := range pollIDs {
		rs, err := vp.store.GetResults(ctx, poolID)
		if err != nil {
			log.Printf("Error getting results for PollID %s: %v", poolID, err)
			continue
		}

		log.Printf("Poll: %s", poolID)
		if len(rs) == 0 {
			log.Println(" No votes counted yet")
			continue
		}
		for optionID, count := range rs {
			log.Printf(" Option %s: %d votes", optionID, count)
		}
	}
	log.Println("-----------------------------")
}
