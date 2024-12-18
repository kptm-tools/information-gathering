package interfaces

import "github.com/kptm-tools/common/common/results"

type IDNSLookupService interface {
	RunScan(targets []string) (*[]results.TargetResult, error)
}

type IDNSLookupHandler interface {
	RunScan() error
}
