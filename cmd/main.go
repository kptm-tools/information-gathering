package main

import (
	"fmt"

	"github.com/kptm-tools/information-gathering/pkg/handlers"
	"github.com/kptm-tools/information-gathering/pkg/services"
)

func main() {
	fmt.Println("Hello information gathering!")

	// Services
	whoIsService := services.NewWhoIsService()
	dnsLookupService := services.NewDNSLookupService()
	harvesterService := services.NewHarvesterService()

	// Handlers
	whoIsHandler := handlers.NewWhoIsHandler(whoIsService)
	dnsLookupHandler := handlers.NewDNSLookupHandler(dnsLookupService)
	harvesterHandler := handlers.NewHarvesterHandler(harvesterService)

	if err := whoIsHandler.RunScan(); err != nil {
		panic(err)
	}

	if err := dnsLookupHandler.RunScan(); err != nil {
		panic(err)
	}

	if err := harvesterHandler.RunScan(); err != nil {
		panic(err)
	}

}
