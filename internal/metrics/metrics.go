package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

/*
Metrics Types:

- CounterVec: A counter with labels. Useful to track things like
  valid vs. duplicate votes per poll.

- Histogram: Tracks the distribution of a value, such as
  processing latency. This lets us see not just the average,
  but also percentiles (e.g. "99% of votes were processed
  in under 10ms").

Registration:
All metrics must be registered with Prometheus during
initialization. Doing this inside a constructor (NewMetrics)
ensures it's only done once.
*/

type ProcessorMetrics struct {
	VotesProcessed *prometheus.CounterVec
	VotesDuplicate *prometheus.CounterVec
	ProcessingTime *prometheus.HistogramVec
}

func NewProcessorMetrics(namespace, subsystem string) *ProcessorMetrics {
	return &ProcessorMetrics{
		VotesProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "votes_processed_total",
				Help:      "Total number of valid votes processed",
			},
			[]string{"poll_id"},
		),
		VotesDuplicate: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "votes_duplicate_total",
				Help:      "Total number of duplicate votes (frauds) detected",
			},
			[]string{"poll_id"},
		),
		ProcessingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "vote_processing_time_seconds",
				Help:      "Histogram of vote processing times",
				Buckets:   prometheus.LinearBuckets(0.001, 0.001, 10), // 10 buckets, 1ms to 10ms
			},
			[]string{"poll_id"},
		),
	}
}
