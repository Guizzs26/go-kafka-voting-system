package model

import "time"

type Vote struct {
	PollID    string    `json:"poll_id"`
	UserID    string    `json:"user_id"`
	OptionID  string    `json:"option_id"`
	Timestamp time.Time `json:"timestamp"`
}
