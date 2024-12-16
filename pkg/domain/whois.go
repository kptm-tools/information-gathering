package domain

import (
	"encoding/json"
	"fmt"

	whoisparser "github.com/likexian/whois-parser"
)

type WhoIsEventResult struct {
	Hosts []whoisparser.WhoisInfo `json:"hosts"`
}

func (w *WhoIsEventResult) String() string {
	data, err := json.MarshalIndent(w, "", " ")
	if err != nil {
		return fmt.Sprintf("Error marshalling WhoisEventResult: %v", err)
	}
	return fmt.Sprintf("WhoIs Event Result\n%s", string(data))
}
