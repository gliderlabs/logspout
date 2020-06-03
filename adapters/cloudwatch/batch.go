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

type Batch struct {
	Msgs []Message
	Size int64
}

// Rules for creating Cloudwatch Log batches, from https://goo.gl/TrIN8c
const MAX_BATCH_COUNT = 10000  // messages
const MAX_BATCH_SIZE = 1048576 // bytes
const MSG_OVERHEAD = 26        // bytes

func msgSize(msg Message) int64 {
	return int64((len(msg.Message) * 8) + MSG_OVERHEAD)
}

func NewBatch() *Batch {
	return &Batch{
		Msgs: []Message{},
		Size: 0,
	}
}

func (b *Batch) Append(msg Message) {
	b.Msgs = append(b.Msgs, msg)
	b.Size = b.Size + msgSize(msg)
}
