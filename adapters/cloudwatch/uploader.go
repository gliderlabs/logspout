package cloudwatch

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

// Uploader receieves CloudwatchBatches on its input channel,
// and sends them on to the AWS Cloudwatch Logs endpoint.
type Uploader struct {
	Input    chan Batch
	svc      *cloudwatchlogs.CloudWatchLogs
	tokens   map[string]string
	debugSet bool
}

// NewUploader creates and returns a new Uploader for the current EC2 Region
func NewUploader(adapter *Adapter) *Uploader {
	region := adapter.Route.Address
	if (region == "auto") || (region == "") {
		if adapter.Ec2Region == "" {
			log.Println("cloudwatch: ERROR - could not get region from EC2")
		} else {
			region = adapter.Ec2Region
		}
	}
	debugSet := false
	_, debugOption := adapter.Route.Options[`DEBUG`]
	if debugOption || (os.Getenv(`DEBUG`) != "") {
		debugSet = true
		log.Println("cloudwatch: Creating AWS Cloudwatch client for region",
			region)
	}
	awsLogLevel := aws.LogOff
	if debugSet {
		awsLogLevel = aws.LogDebugWithRequestRetries
	}
	uploader := Uploader{
		Input:    make(chan Batch),
		tokens:   map[string]string{},
		debugSet: debugSet,
		svc: cloudwatchlogs.New(session.New(),
			&aws.Config{
				Region:     aws.String(region),
				MaxRetries: &adapter.maxRetries,
				LogLevel:   &awsLogLevel,
			}),
	}
	go uploader.Start()
	return &uploader
}

// Start begins the ain loop for the Uploader- POSTs each batch to AWS Cloudwatch
// Logs, while keeping track of the unique sequence token for each log stream.
func (u *Uploader) Start() {
	for batch := range u.Input {
		msg := batch.Msgs[0]
		u.log("Submitting batch for %s-%s (length %d, size %v)",
			msg.Group, msg.Stream, len(batch.Msgs), batch.Size)

		// fetch and cache the upload sequence token
		var token *string
		if cachedToken, isCached := u.tokens[msg.Container]; isCached {
			token = &cachedToken
			u.log("Got token from cache: %s", *token)
		} else {
			u.log("Fetching token from AWS...")
			awsToken, err := u.getSequenceToken(msg)
			if err != nil {
				u.log("ERROR: %s", err)
				continue
			}
			if awsToken != nil {
				u.tokens[msg.Container] = *(awsToken)
				u.log("Got token from AWS: %s", *awsToken)
				token = awsToken
			}
		}

		// generate the array of InputLogEvent from the batch's contents
		events := []*cloudwatchlogs.InputLogEvent{}
		for _, msg := range batch.Msgs {
			event := cloudwatchlogs.InputLogEvent{
				Message:   aws.String(msg.Message),
				Timestamp: aws.Int64(msg.Time.UnixNano() / 1000000),
			}
			events = append(events, &event)
		}
		params := &cloudwatchlogs.PutLogEventsInput{
			LogEvents:     events,
			LogGroupName:  aws.String(msg.Group),
			LogStreamName: aws.String(msg.Stream),
			SequenceToken: token,
		}

		u.log("POSTing PutLogEvents to %s-%s with %d messages, %d bytes",
			msg.Group, msg.Stream, len(batch.Msgs), batch.Size)
		resp, err := u.svc.PutLogEvents(params)
		if err != nil {
			u.log(err.Error())
			u.log("Dropping %d messages", len(events))
			continue
		}
		u.log("Got 200 response")
		if resp.NextSequenceToken != nil {
			u.log("Caching new sequence token for %s-%s: %s",
				msg.Group, msg.Stream, *resp.NextSequenceToken)
			u.tokens[msg.Container] = *resp.NextSequenceToken
		}
	}
}

// AWS CLIENT METHODS

// returns the next sequence token for the log stream associated
// with the given message's group and stream. Creates the stream as needed.
func (u *Uploader) getSequenceToken(msg Message) (*string, error) {
	group, stream := msg.Group, msg.Stream
	groupExists, err := u.groupExists(group)
	if err != nil {
		return nil, err
	}
	if !groupExists {
		err = u.createGroup(group)
		if err != nil {
			return nil, err
		}
	}
	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(group),
		LogStreamNamePrefix: aws.String(stream),
	}
	u.log("Describing stream %s-%s...", group, stream)
	resp, err := u.svc.DescribeLogStreams(params)
	if err != nil {
		return nil, err
	}
	if count := len(resp.LogStreams); count > 1 { // too many matching streams!
		return nil, fmt.Errorf(
			"%d streams match group %s, stream %s", count, group, stream)
	}
	if len(resp.LogStreams) == 0 { // no matching streams - create one and retry
		if err = u.createStream(group, stream); err != nil {
			return nil, err
		}
		token, err := u.getSequenceToken(msg)
		return token, err
	}
	return resp.LogStreams[0].UploadSequenceToken, nil
}

func (u *Uploader) groupExists(group string) (bool, error) {
	u.log("Checking for group: %s...", group)
	resp, err := u.svc.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(group),
	})
	if err != nil {
		return false, err
	}
	for _, matchedGroup := range resp.LogGroups {
		if *matchedGroup.LogGroupName == group {
			return true, nil
		}
	}
	return false, nil
}

func (u *Uploader) createGroup(group string) error {
	u.log("Creating group: %s...", group)
	params := &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(group),
	}
	if _, err := u.svc.CreateLogGroup(params); err != nil {
		return err
	}
	return nil
}

func (u *Uploader) createStream(group, stream string) error {
	u.log("Creating stream for group %s, stream %s...", group, stream)
	params := &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(group),
		LogStreamName: aws.String(stream),
	}
	if _, err := u.svc.CreateLogStream(params); err != nil {
		return err
	}
	return nil
}

// HELPER METHODS

func (u *Uploader) log(format string, args ...interface{}) {
	if u.debugSet {
		msg := fmt.Sprintf(format, args...)
		msg = fmt.Sprintf("cloudwatch: %s", msg)
		if !strings.HasSuffix(msg, "\n") {
			msg = fmt.Sprintf("%s\n", msg)
		}
		log.Print(msg)
	}
}
