package processing

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/metrics"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
)

type VoteProcessor struct {
	consumer event.VoteConsumer
	metrics  *metrics.ProcessorMetrics

	mu             sync.RWMutex
	votesProcessed map[string]map[string]bool // Structure : [pollID][userID] -> bool
	results        map[string]map[string]int  // Structure : [pollID][optionID] -> count
}

func NewVoteProcessor(c event.VoteConsumer, m *metrics.ProcessorMetrics) *VoteProcessor {
	return &VoteProcessor{
		consumer:       c,
		metrics:        m,
		votesProcessed: make(map[string]map[string]bool),
		results:        make(map[string]map[string]int),
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
			vp.printResults()

		default:
			v, err := vp.consumer.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() == context.Canceled {
					continue
				}
				log.Printf("Error reading message: %v", err)
				continue
			}
			vp.processVote(v)
		}
	}
}

func (vp *VoteProcessor) processVote(v model.Vote) {
	start := time.Now()
	defer func() {
		durations := time.Since(start).Seconds()
		vp.metrics.ProcessingTime.WithLabelValues(v.PollID).Observe(durations)
	}()

	vp.mu.Lock()
	defer vp.mu.Unlock()

	if _, ok := vp.votesProcessed[v.PollID]; !ok {
		vp.votesProcessed[v.PollID] = make(map[string]bool)
		vp.results[v.PollID] = make(map[string]int)
	}

	if vp.votesProcessed[v.PollID][v.UserID] {
		log.Printf("[FRAUD DETECTED] Duplicate vote from UserID: %s to PollID: %s", v.UserID, v.PollID)
		vp.metrics.VotesDuplicate.WithLabelValues(v.PollID).Inc()
		return // we're done here
	}

	// If you've made it this far, your vote is valid
	log.Printf("[VALID VOTE] UserID: %s voted for OptionID: %s in PollID: %s", v.UserID, v.OptionID, v.PollID)
	vp.metrics.VotesProcessed.WithLabelValues(v.PollID).Inc()

	vp.votesProcessed[v.PollID][v.UserID] = true
	vp.results[v.PollID][v.OptionID]++
}

func (vp *VoteProcessor) printResults() {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	log.Println("--- CURRENT SCORE ---")
	if len(vp.results) == 0 {
		log.Println("No valid votes counted yet")
	}

	for poolID, options := range vp.results {
		log.Printf("Poll: %s", poolID)
		for optionID, count := range options {
			log.Printf(" -> Option: %s | Votes: %d", optionID, count)
		}
	}
	log.Println("--------------------")
}
