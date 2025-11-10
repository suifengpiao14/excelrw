package excelrw

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/suifengpiao14/apihttpprotocol"
	"github.com/suifengpiao14/excelrw/repository"
	"github.com/suifengpiao14/httpraw"
)

type ProxyRequestIn struct {
	ConfigId         int                `json:"configId"`
	ReqDTO           httpraw.RequestDTO `json:"reqDTO"`
	BusinessCodePath string             `json:"businessCodePath"` //业务成功标识路径，例如：$.code
	BusinessOkCode   string             `json:"businessOkCode"`   //业务成功标识值，例如：0
}

func ProxyRequest(in *ProxyRequestIn) (err error) {
	requestLogService := repository.NewRequestLogRepository()
	reqeustLogInsertIn := repository.RequestLogRepositoryInsertIn{
		ConfigId:         in.ConfigId,
		RequestDTO:       in.ReqDTO.String(),
		BusinessCodePath: in.BusinessCodePath,
		BusinessOkCode:   in.BusinessOkCode,
		CURL:             in.ReqDTO.CurlCommand(),
	}
	id, err := requestLogService.Insert(reqeustLogInsertIn)
	if err != nil {
		return err
	}
	requestDTO := in.ReqDTO

	client := apihttpprotocol.NewClientProtocol(requestDTO.Method, requestDTO.URL)
	client.Request().Headers = requestDTO.Headers.HttpHeaders() //设置头

	var resp json.RawMessage
	err = client.Do(requestDTO.Body, &resp)
	response := client.Response()
	responseDTO := httpraw.ResponseDTO{
		Headers: httpraw.HttpHeader2Headers(response.Headers),
		Body:    string(resp),
	}
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	result := "failure"
	if err == nil {
		result = "success"
	}
	updateIn := repository.RequestLogRepositoryUpdateResponseIn{
		Id:          cast.ToInt(id),
		ResponseDTO: responseDTO.String(),
		Error:       errStr,
		Result:      result,
	}
	err = requestLogService.UpdateResponse(updateIn)
	if err != nil {
		err = errors.WithMessagef(err, "curl:%s", client.Request().CurlCommand())
		return err
	}

	return nil
}
