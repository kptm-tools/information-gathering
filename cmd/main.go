package main

import (
	"fmt"

	"github.com/kptm-tools/information-gathering/pkg/whois"
)

func main() {
	fmt.Println("Hello information gathering!")

	targets := []string{"whois.verisign-grs.com"}
	whois.RunWhoIsScan(targets)
}
