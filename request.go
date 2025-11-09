package excelrw

import (
	"github.com/suifengpiao14/excelrw/repository"
	"github.com/suifengpiao14/httpraw"
)

type ProxyRequestIn struct {
	ReqDTO           httpraw.RequestDTO `json:"reqDTO"`
	BusinessCodePath string             `json:"businessCodePath"` //业务成功标识路径，例如：$.code
	BusinessOkCode   string             `json:"businessOkCode"`   //业务成功标识值，例如：0
}

func ProxyRequest(req *ProxyRequestIn) (err error) {
	requestLogService := repository.NewRequestLogRepository()
	reqeustLogInsertIn := repository.RequestLogRepositoryInsertIn{
		RequestDTO: req.ReqDTO.String(),
	}
	requestLogService.Insert(repository.RequestLogRepositoryInsertIn{})
	return nil
}
