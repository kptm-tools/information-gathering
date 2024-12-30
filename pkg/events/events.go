package events

import (
	"encoding/json"
	"log"

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

		log.Printf("Received ScanStarted Event\n")
		// 1. Parse the message payload
		var payload cmmn.ScanStartedEvent

		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Printf("Received invalid JSON payload: %s\n", msg.Data)
			// 1.1 Publish scan failed
			return
		}
		log.Printf("Payload: %+v\n", payload)

		// 2. Call our handlers for each tool
		err := whoIsHandler.RunScan()
		if err != nil {
			log.Printf("Error on WhoIsHandler: %s", err.Error())
			// Publish the result
		}

		err = dnsLookupHandler.RunScan()
		if err != nil {
			log.Printf("Error on WhoIsHandler: %s", err.Error())
			// Publish the result
		}

		err = harvesterHandler.RunScan()
		if err != nil {
			log.Printf("Error on HarvesterHandler: %s", err.Error())
			// Publish the result
		}

	})

	return nil

}
