package cloudwatch

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gliderlabs/logspout/router"
)

const DEFAULT_DELAY = 4 //seconds

// CloudwatchBatcher receieves Cloudwatch messages on its input channel,
// stores them in CloudwatchBatches until enough data is ready to send, then
// sends each CloudwatchMessageBatch on its output channel.
type CloudwatchBatcher struct {
	Input  chan CloudwatchMessage
	output chan CloudwatchBatch
	route  *router.Route
	timer  chan bool
	// maintain a batch for each container, indexed by its name
	batches map[string]*CloudwatchBatch
}

// constructor for CloudwatchBatcher - requires the adapter
func NewCloudwatchBatcher(adapter *CloudwatchAdapter) *CloudwatchBatcher {
	batcher := CloudwatchBatcher{
		Input:   make(chan CloudwatchMessage),
		output:  NewCloudwatchUploader(adapter).Input,
		batches: map[string]*CloudwatchBatch{},
		timer:   make(chan bool),
		route:   adapter.Route,
	}
	go batcher.Start()
	return &batcher
}

// Main loop for the Batcher - just sorts each messages into a batch, but
// submits the batch first and replaces it if the message is too big.
func (b *CloudwatchBatcher) Start() {
	go b.RunTimer()
	for { // run forever, and...
		select { // either batch up a message, or respond to the timer
		case msg := <-b.Input: // a message - put it into its slice
			if len(msg.Message) == 0 { // empty messages are not allowed
				break
			}
			// get or create the correct slice of messages for this message
			if _, exists := b.batches[msg.Container]; !exists {
				b.batches[msg.Container] = NewCloudwatchBatch()
			}
			// if Msg is too long for the current batch, submit the batch
			if (b.batches[msg.Container].Size+msgSize(msg)) > MAX_BATCH_SIZE ||
				len(b.batches[msg.Container].Msgs) >= MAX_BATCH_COUNT {
				b.output <- *b.batches[msg.Container]
				b.batches[msg.Container] = NewCloudwatchBatch()
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

func (b *CloudwatchBatcher) RunTimer() {
	delayText := strconv.Itoa(DEFAULT_DELAY)
	if routeDelay, isSet := b.route.Options[`DELAY`]; isSet {
		delayText = routeDelay
	}
	if envDelay := os.Getenv(`DELAY`); envDelay != "" {
		delayText = envDelay
	}
	delay, err := strconv.Atoi(delayText)
	if err != nil {
		log.Printf("WARNING: ERROR parsing DELAY %s, using default of %d\n",
			delayText, DEFAULT_DELAY)
		delay = DEFAULT_DELAY
	}
	for {
		time.Sleep(time.Duration(delay) * time.Second)
		b.timer <- true
	}
}
