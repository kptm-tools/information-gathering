package interfaces

import "github.com/kptm-tools/information-gathering/pkg/domain"

type IWhoIsService interface {
	RunScan(targets []string) (*domain.WhoIsEventResult, error)
}

type IWhoIsHandler interface {
	RunScan() error
}
