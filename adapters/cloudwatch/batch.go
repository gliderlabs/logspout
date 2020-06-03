package cloudwatch

import "time"

// Message is a simple JSON input to Cloudwatch.
type Message struct {
	Message   string    `json:"message"`
	Group     string    `json:"group"`
	Stream    string    `json:"stream"`
	Time      time.Time `json:"time"`
	Container string    `json:"container"`
}

// Batch is a group of Messages to be submitted to Cloudwatch
// as part of a single request
type Batch struct {
	Msgs []Message
	Size int64
}

const msgOverhead = 26 // bytes

func msgSize(msg Message) int64 {
	return int64((len(msg.Message) * 8) + msgOverhead)
}

// NewBatch creates and returns an empty Batch
func NewBatch() *Batch {
	return &Batch{
		Msgs: []Message{},
		Size: 0,
	}
}

// Append adds Messages to a Batch
func (b *Batch) Append(msg Message) {
	b.Msgs = append(b.Msgs, msg)
	b.Size = b.Size + msgSize(msg)
}
