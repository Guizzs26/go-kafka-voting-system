package processing

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Guizzs26/real_time_voting_analysis_system/internal/event"
	"github.com/Guizzs26/real_time_voting_analysis_system/internal/model"
)

/*

A missão dele é usar a interface VoteConsumer que criamos
para buscar os votos e aplicar nossas regras:

Receber um voto.

Verificar se já vimos esse par (PollID, UserID) antes.

Se for novo, computar o voto e marcá-lo como "processado".

Se for duplicado, descartá-lo.

Periodicamente, mostrar os resultados agregados.

*/

type VoteProcessor struct {
	consumer event.VoteConsumer

	mu             sync.RWMutex
	votesProcessed map[string]map[string]bool // Estrutura: [pollID][userID] -> bool
	results        map[string]map[string]int  // Estrutura: [pollID][optionID] -> count
}

func NewVoteProcessor(c event.VoteConsumer) *VoteProcessor {
	return &VoteProcessor{
		consumer:       c,
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
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if _, ok := vp.votesProcessed[v.PollID]; !ok {
		vp.votesProcessed[v.PollID] = make(map[string]bool)
		vp.results[v.PollID] = make(map[string]int)
	}

	if vp.votesProcessed[v.PollID][v.UserID] {
		log.Printf("[FRAUD DETECTED] Duplicate vote from UserID: %s to PollID: %s", v.UserID, v.PollID)
		return // we're done here
	}

	// If you've made it this far, your vote is valid
	log.Printf("[VALID VOTE] UserID: %s voted for OptionID: %s in PollID: %s", v.UserID, v.OptionID, v.PollID)

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
