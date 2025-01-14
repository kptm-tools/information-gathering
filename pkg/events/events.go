package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kptm-tools/common/common/enums"
	"github.com/kptm-tools/common/common/events"
	cmmn "github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
	"github.com/nats-io/nats.go"
)

func SubscribeToScanStarted(
	bus cmmn.EventBus,
	whoIsHandler interfaces.IWhoIsHandler,
	dnsLookupHandler interfaces.IDNSLookupHandler,
	harvesterHandler interfaces.IHarvesterHandler,
) error {

	bus.Subscribe("ScanStarted", func(msg *nats.Msg) {

		go func(msg *nats.Msg) {

			log.Printf("Received ScanStarted Event\n")
			// 1. Parse the message payload
			var payload cmmn.ScanStartedEvent

			if err := json.Unmarshal(msg.Data, &payload); err != nil {
				log.Printf("Received invalid JSON payload: %s\n", msg.Data)
				// 1.1 Publish scan failed
				msg, err := json.Marshal(map[string]string{"reason": "Invalid JSON payload", "payload": string(msg.Data)})
				if err != nil {
					log.Printf("failed to marshal scan failed payload: %v", err)
				}
				bus.Publish("ScanFailed", msg)
				return
			}
			log.Printf("Payload: %+v\n", payload)

			// TODO: Implement context for cancellation
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// 2. Call our handlers for each tool
			c := fanIn(
				whoIsHandler.RunScan(ctx, payload),
				dnsLookupHandler.RunScan(ctx, payload),
				harvesterHandler.RunScan(ctx, payload),
			)

			for result := range c {
				processServiceResult(result, bus)
			}
			log.Printf("Finished gathering information for Scan %s!\n", payload.ScanID)
		}(msg)

	})

	return nil

}

func fanIn(inputs ...<-chan interfaces.ServiceResult) <-chan interfaces.ServiceResult {
	c := make(chan interfaces.ServiceResult)
	var wg sync.WaitGroup

	for _, input := range inputs {
		wg.Add(1)
		go func(ch <-chan interfaces.ServiceResult) {
			defer wg.Done()
			for result := range ch {
				c <- result
			}
		}(input)
	}

	// Close the channel when wg is done
	go func() {
		wg.Wait()
		close(c)
	}()
	return c
}

func processServiceResult(result interfaces.ServiceResult, bus cmmn.EventBus) {
	// 3. When each one finishes, it must publish it's event
	subject, err := getSubjectName(result.ServiceName)
	if err != nil {
		log.Printf("failed to find subject name: %v", err)
		return
	}
	if result.Err != nil {
		log.Printf("Error posting to %s: %v\n", subject, result.Err)
		return
	}

	// TODO: Make this its own method, then wrap both methods in a publish function
	log.Printf("Posting result to %s: %+v\n", subject, result)
	payload, err := buildEventPayload(result)
	if err != nil {
		log.Printf("failed to build event payload for subject %s: %v\n", subject, err)
		return
	}

	bus.Publish(subject, payload)

}

func getSubjectName(serviceName enums.ServiceName) (string, error) {
	subjectNameMap := map[enums.ServiceName]enums.EventSubjectName{
		enums.ServiceWhoIs:     enums.WhoIsEventSubject,
		enums.ServiceDNSLookup: enums.DNSLookupEventSubject,
		enums.ServiceHarvester: enums.HarvesterEventSubject,
	}

	subject, exists := subjectNameMap[serviceName]
	if !exists {
		return "", fmt.Errorf("invalid service: %s", serviceName)
	}
	return string(subject), nil

}

func buildEventPayload(result interfaces.ServiceResult) ([]byte, error) {
	var (
		msg []byte
		err error
	)
	timestamp := time.Now().Unix()
	eventType, exists := cmmn.ServiceEventMap[result.ServiceName]
	if !exists {
		return nil, fmt.Errorf("invalid service: %s", result.ServiceName)
	}

	// Handle the error attribute safely
	var eventError *cmmn.EventError
	if result.Err != nil {
		eventError = &cmmn.EventError{
			Code:    result.Err.Error(),
			Message: result.Err.Error(),
		}
	}

	baseEvt := cmmn.BaseEvent{
		ScanID:    result.ScanID,
		Error:     eventError,
		Timestamp: timestamp,
	}

	switch v := eventType.(type) {
	case events.WhoIsEvent:
		evt := cmmn.WhoIsEvent{
			BaseEvent: baseEvt,
			Results:   result.Result,
		}
		msg, err = json.Marshal(evt)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal WhoIsEvent: %w", err)
		}
	case events.DNSLookupEvent:
		evt := cmmn.DNSLookupEvent{
			BaseEvent: baseEvt,
			Results:   result.Result,
		}
		msg, err = json.Marshal(evt)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal DNSLookupEvent: %w", err)
		}
	case events.HarvesterEvent:
		evt := cmmn.HarvesterEvent{
			BaseEvent: baseEvt,
			Results:   result.Result,
		}
		msg, err = json.Marshal(evt)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal HarvesterEvent: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown event type: %T", v)
	}

	return msg, nil

}
