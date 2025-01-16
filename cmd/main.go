package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	cmmn "github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/information-gathering/pkg/config"
	"github.com/kptm-tools/information-gathering/pkg/events"
	"github.com/kptm-tools/information-gathering/pkg/handlers"
	"github.com/kptm-tools/information-gathering/pkg/services"
	"github.com/lmittmann/tint"
)

func main() {
	fmt.Println("Hello information gathering!")
	c := config.LoadConfig()

	// Logger
	w := os.Stdout
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.Stamp,
		}),
	))

	// Events
	eventBus, err := cmmn.NewNatsEventBus(c.GetNatsConnStr())
	if err != nil {
		slog.Error("Error creating Event Bus",
			slog.Any("error", err),
			slog.String("NatsConnStr", c.GetNatsConnStr()),
		)
		panic(err)
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
			slog.Error("failed to subscribe to ScanStartedEvent", slog.Any("error", err))
			return err
		}
		if err := events.SubscribeToScanCancelled(eventBus); err != nil {
			slog.Error("failed to subscribe to ScanCancelledEvent", slog.Any("error", err))
			return err
		}
		return nil
	})
	if err != nil {
		slog.Error("Failed to initialize Event Bus:", slog.Any("error", err))
		panic(err)
	}

	waitForShutdown()

}

func waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-stop
	slog.Info("Shutting down gracefully...")
}
