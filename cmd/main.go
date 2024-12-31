package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	cmmn "github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/information-gathering/pkg/config"
	"github.com/kptm-tools/information-gathering/pkg/events"
	"github.com/kptm-tools/information-gathering/pkg/handlers"
	"github.com/kptm-tools/information-gathering/pkg/services"
)

func main() {
	fmt.Println("Hello information gathering!")
	c := config.LoadConfig()

	// Events
	eventBus, err := cmmn.NewNatsEventBus(c.GetNatsConnStr())
	if err != nil {
		log.Fatalf("Error creating Event Bus: %s", err.Error())
	}

	// Services
	whoIsService := services.NewWhoIsService()
	dnsLookupService := services.NewDNSLookupService()
	harvesterService := services.NewHarvesterService()

	// Handlers
	whoIsHandler := handlers.NewWhoIsHandler(whoIsService)
	dnsLookupHandler := handlers.NewDNSLookupHandler(dnsLookupService)
	harvesterHandler := handlers.NewHarvesterHandler(harvesterService)

	err = eventBus.Init(func() error {
		if err := events.SubscribeToScanStarted(eventBus, whoIsHandler, dnsLookupHandler, harvesterHandler); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to initialize Event Bus: %s", err.Error())
	}

	waitForShutdown()

}

func waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-stop
	fmt.Println("Shutting down gracefully...")
}
