package interfaces

import cmmn "github.com/kptm-tools/common/common/results"

type ServiceResult struct {
	ScanID      string
	ServiceName cmmn.ServiceName
	Result      []cmmn.TargetResult
	Err         error
}
