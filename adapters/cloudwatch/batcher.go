package cloudwatch

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gliderlabs/logspout/router"
)

const DefaultDelay = 4 //seconds

// Batcher receieves Cloudwatch messages on its input channel,
// stores them in CloudwatchBatches until enough data is ready to send, then
// sends each CloudwatchMessageBatch on its output channel.
type Batcher struct {
	Input  chan Message
	output chan Batch
	route  *router.Route
	timer  chan bool
	// maintain a batch for each container, indexed by its name
	batches map[string]*Batch
}

// constructor for CloudwatchBatcher - requires the adapter
func NewCloudwatchBatcher(adapter *Adapter) *Batcher {
	batcher := Batcher{
		Input:   make(chan Message),
		output:  NewUploader(adapter).Input,
		batches: map[string]*Batch{},
		timer:   make(chan bool),
		route:   adapter.Route,
	}
	go batcher.Start()
	return &batcher
}

// Main loop for the Batcher - just sorts each messages into a batch, but
// submits the batch first and replaces it if the message is too big.
func (b *Batcher) Start() {
	go b.RunTimer()
	for { // run forever, and...
		select { // either batch up a message, or respond to the timer
		case msg := <-b.Input: // a message - put it into its slice
			if len(msg.Message) == 0 { // empty messages are not allowed
				break
			}
			// get or create the correct slice of messages for this message
			if _, exists := b.batches[msg.Container]; !exists {
				b.batches[msg.Container] = NewBatch()
			}
			// if Msg is too long for the current batch, submit the batch
			if (b.batches[msg.Container].Size+msgSize(msg)) > MaxBatchSize ||
				len(b.batches[msg.Container].Msgs) >= MaxBatchCount {
				b.output <- *b.batches[msg.Container]
				b.batches[msg.Container] = NewBatch()
			}
			thisBatch := b.batches[msg.Container]
			thisBatch.Append(msg)
		case <-b.timer: // submit and delete all existing batches
			for container, batch := range b.batches {
				b.output <- *batch
				delete(b.batches, container)
			}
		}
	}
}

func (b *Batcher) RunTimer() {
	delayText := strconv.Itoa(DefaultDelay)
	if routeDelay, isSet := b.route.Options[`DELAY`]; isSet {
		delayText = routeDelay
	}
	if envDelay := os.Getenv(`DELAY`); envDelay != "" {
		delayText = envDelay
	}
	delay, err := strconv.Atoi(delayText)
	if err != nil {
		log.Printf("WARNING: ERROR parsing DELAY %s, using default of %d\n",
			delayText, DefaultDelay)
		delay = DefaultDelay
	}
	for {
		time.Sleep(time.Duration(delay) * time.Second)
		b.timer <- true
	}
}
