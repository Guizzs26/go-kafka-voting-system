package processing

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/metrics"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/pubsub"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/store"
)

type VoteProcessor struct {
	consumer   event.VoteConsumer
	publisher  event.VotePublisher
	metrics    *metrics.ProcessorMetrics
	store      store.VoteStore
	hub        *pubsub.Hub
	numWorkers int
	wg         sync.WaitGroup

	// we maintain minimal, local state: just the IDs of polls we've already seen
	mu         sync.Mutex
	knownPolls map[string]bool
}

func NewVoteProcessor(
	c event.VoteConsumer,
	p event.VotePublisher,
	m *metrics.ProcessorMetrics,
	s store.VoteStore,
	h *pubsub.Hub,
	nw int,
) *VoteProcessor {
	return &VoteProcessor{
		consumer:   c,
		publisher:  p,
		metrics:    m,
		store:      s,
		hub:        h,
		numWorkers: nw,
		knownPolls: make(map[string]bool),
	}
}

func (vp *VoteProcessor) Run(ctx context.Context) error {
	rTicker := time.NewTicker(5 * time.Second)
	defer rTicker.Stop()

	jobs := make(chan model.Vote, 100)
	for i := 1; i < vp.numWorkers; i++ {
		vp.wg.Add(1)
		go vp.worker(ctx, i+1, jobs)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Message reader got stop signal")
				close(jobs)
				return

			default:
				v, err := vp.consumer.ReadMessage(ctx)
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
						continue
					}
					log.Printf("Error reading message from kafka: %v", err)
					continue
				}
				jobs <- v
			}
		}
	}()

	resultsTicker := time.NewTicker(5 * time.Second)
	defer resultsTicker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Results reporter got stop signal")
				return
			case <-resultsTicker.C:
				vp.printResults(ctx)
			}
		}
	}()

	log.Println("Vote processor and workers started")
	<-ctx.Done()
	log.Println("Vote processor shutting down, waiting for workers...")

	vp.wg.Wait()
	log.Println("All workers finished")
	return nil
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

	r, err := vp.store.GetResults(ctx, v.PollID)
	if err != nil {
		log.Printf("Error getting results for PollID %s: %v", v.PollID, err)
		return
	}

	rJSON, err := json.Marshal(r)
	if err != nil {
		log.Printf("Error marshalling vote to JSON: %v", err)
		return
	}

	m := &pubsub.Message{
		PollID: v.PollID,
		Data:   rJSON,
	}

	select {
	case vp.hub.Broadcast <- m:
		// message sent successfully
		log.Printf("Poll score %s sent to Hub.", v.PollID)
	default:
		log.Printf("Warning: Broadcast channel is full, dropping message for PollID: %s", v.PollID)
	}
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

func (vp *VoteProcessor) worker(ctx context.Context, id int, jobs <-chan model.Vote) {
	defer vp.wg.Done()
	log.Printf("Worker %d started", id)

	for vote := range jobs {
		vp.processVote(ctx, vote)
	}

	log.Printf("Worker %d finished", id)
}
