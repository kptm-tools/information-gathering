package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	cmmn "github.com/kptm-tools/common/common/events"
	res "github.com/kptm-tools/common/common/results"
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

		services := map[res.ServiceName]func(cmmn.ScanStartedEvent) ([]res.TargetResult, error){
			res.ServiceWhoIs:     whoIsHandler.RunScan,
			res.ServiceDNSLookup: dnsLookupHandler.RunScan,
			res.ServiceHarvester: harvesterHandler.RunScan,
		}

		log.Printf("Received ScanStarted Event\n")
		// 1. Parse the message payload
		var payload cmmn.ScanStartedEvent

		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Printf("Received invalid JSON payload: %s\n", msg.Data)
			// 1.1 Publish scan failed
			return
		}
		log.Printf("Payload: %+v\n", payload)

		results := make(chan ServiceResult)
		ctx, cancel := context.WithCancel(context.Background())
		// 2. Call our handlers for each tool
		for name, service := range services {
			go task(ctx, name, service, payload, results)
		}

		go func() {
			time.Sleep(3 * time.Second)
			log.Println("Cancelling remaining tasks...")
			cancel()
		}()

		// 3. When each one finishes, it must publish it's event
		for i := 0; i < len(services); i++ {
			select {
			case r := <-results:
				if r.Err != nil {
					log.Printf("Error from %s: %v\n", r.ServiceName, r.Err)
				} else {
					log.Println("Results from", r.ServiceName)
					for _, result := range r.Result {
						log.Println(result.String())
					}
				}

			case <-ctx.Done():
				log.Println("Context cancelled. Stopping collection")
				return
			}
		}

	})

	return nil

}

type ServiceResult struct {
	ScanID      string
	ServiceName res.ServiceName
	Result      []res.TargetResult
	Err         error
}

func task(ctx context.Context, taskName res.ServiceName, task func(cmmn.ScanStartedEvent) ([]res.TargetResult, error), evt cmmn.ScanStartedEvent, results chan<- ServiceResult) {
	select {
	case <-ctx.Done():
		// Handle cancellation
		results <- ServiceResult{
			ScanID:      evt.ScanID,
			ServiceName: taskName,
			Result:      []res.TargetResult{},
			Err:         fmt.Errorf("service %s cancelled", taskName),
		}
	default:
		// Execute the actual task
		result, err := task(evt)
		results <- ServiceResult{
			ServiceName: taskName,
			Result:      result,
			Err:         err,
		}
	}

}
