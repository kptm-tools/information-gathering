package interfaces

import "github.com/kptm-tools/common/common/results"

type IWhoIsService interface {
	RunScan(targets []string) (*[]results.TargetResult, error)
}

type IWhoIsHandler interface {
	RunScan() error
}
