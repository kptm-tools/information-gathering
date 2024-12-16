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

	// Handlers
	whoIsHandler := handlers.NewWhoIsHandler(whoIsService)

	err := whoIsHandler.RunScan()
	if err != nil {
		panic(err)
	}

}
