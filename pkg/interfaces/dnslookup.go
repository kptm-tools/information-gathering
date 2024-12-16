package interfaces

import "github.com/kptm-tools/information-gathering/pkg/domain"

type IDNSLookupService interface {
	RunScan(targets []string) (*domain.DNSLookupEventResult, error)
}

type IDNSLookupHandler interface {
	RunScan() error
}
