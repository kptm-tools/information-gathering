package interfaces

import (
	"github.com/kptm-tools/common/common/enums"
	cmmn "github.com/kptm-tools/common/common/results"
)

type ServiceResult struct {
	ScanID      string
	ServiceName enums.ServiceName
	Result      []cmmn.TargetResult
	Err         error
}
