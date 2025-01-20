package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kptm-tools/common/common/enums"
	cmmn "github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
	"github.com/nats-io/nats.go"
)

var scanContextMap sync.Map

func SubscribeToScanStarted(
	bus cmmn.EventBus,
	whoIsHandler interfaces.IWhoIsHandler,
	dnsLookupHandler interfaces.IDNSLookupHandler,
	harvesterHandler interfaces.IHarvesterHandler,
) error {

	bus.Subscribe(string(enums.ScanStartedEventSubject), func(msg *nats.Msg) {

		go func(msg *nats.Msg) {

			slog.Info("Received ScanStarted Event\n")
			// 1. Parse the message payload
			var payload cmmn.ScanStartedEvent

			if err := json.Unmarshal(msg.Data, &payload); err != nil {
				slog.Error("Received invalid JSON payload",
					slog.String("payload", string(msg.Data)),
					slog.Any("error", err))
				// 1.1 Publish scan failed
				failedPayload := cmmn.NewScanFailedEvent(payload.ScanID, enums.ParsingError, err.Error())
				msg, err := json.Marshal(failedPayload)
				if err != nil {
					slog.Error("failed to marshal scan failed payload", slog.Any("error", err))
					return
				}
				bus.Publish(string(enums.ScanFailedEventSubject), msg)
				return
			}
			slog.Debug("Parsed payload", slog.Any("payload", payload))

			// Cancellation context
			ctx, cancel := context.WithCancel(context.Background())
			scanContextMap.Store(payload.ScanID, cancel)
			defer func() {
				scanContextMap.Delete(payload.ScanID)
				cancel()
			}()

			// 2. Call our handlers for each tool
			c := fanIn(
				whoIsHandler.RunScan(ctx, payload),
				dnsLookupHandler.RunScan(ctx, payload),
				harvesterHandler.RunScan(ctx, payload),
			)

			for result := range c {
				// Publish scan failed if there was an error processing service result
				if err := processServiceResult(result, bus); err != nil {
					slog.Error("Error processing result", slog.Any("error", err))
					failedPayload := cmmn.NewScanFailedEvent(payload.ScanID, enums.ServiceError, err.Error())
					msg, err := json.Marshal(failedPayload)
					if err != nil {
						slog.Error("failed to marshal scan failed payload", slog.Any("error", err))
					}
					bus.Publish(string(enums.ScanFailedEventSubject), msg)
				}
			}
			slog.Info("Finished gathering information", slog.String("scanID", payload.ScanID))
		}(msg)

	})

	return nil

}

func SubscribeToScanCancelled(bus cmmn.EventBus) error {
	bus.Subscribe(string(enums.ScanCancelledEventSubject), func(msg *nats.Msg) {
		go func(msg *nats.Msg) {
			slog.Info("Received ScanCancelledEvent")
			// 1. Parse the message payload
			var payload cmmn.ScanCancelledEvent
			if err := json.Unmarshal(msg.Data, &payload); err != nil {
				slog.Error("Received invalid JSON payload", slog.Any("msgData", msg.Data))
				// 1.1 Publish scan failed
				failedPayload := cmmn.NewScanFailedEvent(payload.ScanID, enums.ServiceError, fmt.Errorf("Invalid JSON payload: %w", err).Error())
				msg, err := json.Marshal(failedPayload)
				if err != nil {
					slog.Error("Failed to marshal scan failed payload", slog.Any("error", err))
				}
				bus.Publish(string(enums.ScanFailedEventSubject), msg)
				return
			}

			slog.Debug("Event payload", slog.Any("payload", payload))
			slog.Info("Cancelling Scan", slog.String("scanID", payload.ScanID))
			if cancelFunc, ok := scanContextMap.Load(payload.ScanID); ok {
				cancelFunc.(context.CancelFunc)() // Cancel the context
				scanContextMap.Delete(payload.ScanID)
				slog.Info("Scan successfully cancelled", slog.String("scanID", payload.ScanID))
			} else {
				slog.Warn("No active scan found for ScanID", slog.String("scanID", payload.ScanID))
			}
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

func processServiceResult(result interfaces.ServiceResult, bus cmmn.EventBus) error {
	// 3. When each one finishes, it must publish it's event
	subject, err := getSubjectName(result.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to find subject name: %w", err)
	}
	if result.Err != nil {
		return fmt.Errorf("failed to post to subject %s: %w", subject, result.Err)
	}

	slog.Info("Publishing service result", slog.String("subject", subject), slog.Any("result", result))
	payload, err := buildEventPayload(result)
	if err != nil {
		return fmt.Errorf("failed to build event payload for subject %s: %w", subject, err)
	}

	if err := bus.Publish(subject, payload); err != nil {
		return fmt.Errorf("failed to publish event payload to subject %s: %w", subject, err)
	}
	return nil

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
			Code:    enums.ServiceError,
			Message: result.Err.Error(),
		}
	}

	baseEvt := cmmn.BaseEvent{
		ScanID:    result.ScanID,
		Error:     eventError,
		Timestamp: timestamp,
	}

	switch v := eventType.(type) {
	case cmmn.WhoIsEvent:
		evt := cmmn.WhoIsEvent{
			BaseEvent: baseEvt,
			Results:   result.Result,
		}
		msg, err = json.Marshal(evt)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal WhoIsEvent: %w", err)
		}
	case cmmn.DNSLookupEvent:
		evt := cmmn.DNSLookupEvent{
			BaseEvent: baseEvt,
			Results:   result.Result,
		}
		msg, err = json.Marshal(evt)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal DNSLookupEvent: %w", err)
		}
	case cmmn.HarvesterEvent:
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
