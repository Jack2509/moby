package awslogs // import "github.com/docker/docker/daemon/logger/awslogs"

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type mockcwlogsclient struct {
	createLogGroupArgument  chan *cloudwatchlogs.CreateLogGroupInput
	createLogGroupResult    chan *createLogGroupResult
	createLogStreamArgument chan *cloudwatchlogs.CreateLogStreamInput
	createLogStreamResult   chan *createLogStreamResult
	putLogEventsArgument    chan *cloudwatchlogs.PutLogEventsInput
	putLogEventsResult      chan *putLogEventsResult
}

type createLogGroupResult struct {
	successResult *cloudwatchlogs.CreateLogGroupOutput
	errorResult   error
}

type createLogStreamResult struct {
	successResult *cloudwatchlogs.CreateLogStreamOutput
	errorResult   error
}

type putLogEventsResult struct {
	successResult *cloudwatchlogs.PutLogEventsOutput
	errorResult   error
}

func newMockClient() *mockcwlogsclient {
	return &mockcwlogsclient{
		createLogGroupArgument:  make(chan *cloudwatchlogs.CreateLogGroupInput, 1),
		createLogGroupResult:    make(chan *createLogGroupResult, 1),
		createLogStreamArgument: make(chan *cloudwatchlogs.CreateLogStreamInput, 1),
		createLogStreamResult:   make(chan *createLogStreamResult, 1),
		putLogEventsArgument:    make(chan *cloudwatchlogs.PutLogEventsInput, 1),
		putLogEventsResult:      make(chan *putLogEventsResult, 1),
	}
}

func newMockClientBuffered(buflen int) *mockcwlogsclient {
	return &mockcwlogsclient{
		createLogStreamArgument: make(chan *cloudwatchlogs.CreateLogStreamInput, buflen),
		createLogStreamResult:   make(chan *createLogStreamResult, buflen),
		putLogEventsArgument:    make(chan *cloudwatchlogs.PutLogEventsInput, buflen),
		putLogEventsResult:      make(chan *putLogEventsResult, buflen),
	}
}

func (m *mockcwlogsclient) CreateLogGroup(input *cloudwatchlogs.CreateLogGroupInput) (*cloudwatchlogs.CreateLogGroupOutput, error) {
	m.createLogGroupArgument <- input
	output := <-m.createLogGroupResult
	return output.successResult, output.errorResult
}

func (m *mockcwlogsclient) CreateLogStream(input *cloudwatchlogs.CreateLogStreamInput) (*cloudwatchlogs.CreateLogStreamOutput, error) {
	m.createLogStreamArgument <- input
	output := <-m.createLogStreamResult
	return output.successResult, output.errorResult
}

func (m *mockcwlogsclient) PutLogEvents(input *cloudwatchlogs.PutLogEventsInput) (*cloudwatchlogs.PutLogEventsOutput, error) {
	events := make([]*cloudwatchlogs.InputLogEvent, len(input.LogEvents))
	copy(events, input.LogEvents)
	m.putLogEventsArgument <- &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     events,
		SequenceToken: input.SequenceToken,
		LogGroupName:  input.LogGroupName,
		LogStreamName: input.LogStreamName,
	}

	// Intended mock output
	output := <-m.putLogEventsResult

	// Checked enforced limits in mock
	totalBytes := 0
	for _, evt := range events {
		if evt.Message == nil {
			continue
		}
		eventBytes := len([]byte(*evt.Message))
		if eventBytes > maximumBytesPerEvent {
			// exceeded per event message size limits
			return nil, fmt.Errorf("maximum bytes per event exceeded: Event too large %d, max allowed: %d", eventBytes, maximumBytesPerEvent)
		}
		// total event bytes including overhead
		totalBytes += eventBytes + perEventBytes
	}

	if totalBytes > maximumBytesPerPut {
		// exceeded per put maximum size limit
		return nil, fmt.Errorf("maximum bytes per put exceeded: Upload too large %d, max allowed: %d", totalBytes, maximumBytesPerPut)
	}

	return output.successResult, output.errorResult
}

type mockmetadataclient struct {
	regionResult chan *regionResult
}

type regionResult struct {
	successResult string
	errorResult   error
}

func newMockMetadataClient() *mockmetadataclient {
	return &mockmetadataclient{
		regionResult: make(chan *regionResult, 1),
	}
}

func (m *mockmetadataclient) Region() (string, error) {
	output := <-m.regionResult
	return output.successResult, output.errorResult
}
