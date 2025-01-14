package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

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
			_, cancel := context.WithCancel(context.Background())
			defer cancel()

			// 2. Call our handlers for each tool
			c := fanIn(
				whoIsHandler.RunScan(payload),
				dnsLookupHandler.RunScan(payload),
				harvesterHandler.RunScan(payload),
			)

			for result := range c {
				processServiceResult(result)
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

func processServiceResult(result interfaces.ServiceResult) {
	// 3. When each one finishes, it must publish it's event
	subject := fmt.Sprintf("subject.%s", result.ServiceName)
	if result.Err != nil {
		log.Printf("Error posting to %s: %v\n", subject, result.Err)
		return
	}
	log.Printf("Posting result to %s: %+v\n", subject, result)

}
