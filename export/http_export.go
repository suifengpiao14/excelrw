package export

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/suifengpiao14/excelrw"
	"github.com/suifengpiao14/httpraw"
)

type HttpExportIn struct {
	Tpl             string
	Script          string
	CallbackTpl     string
	CallbackScript  string
	ExcelFilename   string
	FieldMetas      excelrw.FieldMetas
	RequestInterval time.Duration // 循环请求获取数据的间隔时间
	Timeout         time.Duration // 任务处理最长时间
}

type HttpExport struct {
	proxy           *httpraw.HttpProxy
	excelWriteChan  *excelrw.ExcelChanWriter
	startRowNumber  int
	callbackProxy   *httpraw.HttpProxy
	Timeout         time.Duration // 任务处理最长时间
	context         context.Context
	RequestInterval time.Duration // 循环请求获取数据的间隔时间
	CancelFunc      context.CancelFunc
}

var MaxLoopTimes = 100000 // 最大循环次数，超过这个次数退出循环，并抛出错误

func NewHttpExport(exportIn HttpExportIn, option *excelrw.ExcelChanWriterOption) (ex *HttpExport, err error) {
	ex = &HttpExport{
		Timeout: exportIn.Timeout,
		context: context.Background(),
	}
	if exportIn.RequestInterval > 0 {
		timeoutErr := errors.Errorf("proxy export time out")
		ex.context, ex.CancelFunc = context.WithTimeoutCause(ex.context, exportIn.Timeout, timeoutErr)
	}

	sheet := "sheet1"
	ecw, beginRowNumber, err := excelrw.NewExcelChanWriter(ex.context, exportIn.ExcelFilename, sheet, exportIn.FieldMetas, option)
	if err != nil {
		return nil, err
	}

	proxy, err := httpraw.NewHttpProxy(exportIn.Tpl, exportIn.Script)
	if err != nil {
		return nil, err
	}
	ex.proxy = proxy
	ex.excelWriteChan = ecw
	ex.startRowNumber = beginRowNumber

	if exportIn.CallbackTpl != "" { // 增加回调配置
		ex.callbackProxy, err = httpraw.NewHttpProxy(exportIn.CallbackTpl, exportIn.CallbackScript)
		if err != nil {
			return nil, err
		}
	}

	return ex, nil
}

func (ex *HttpExport) Run(requestParams map[string]any, callback func(ctx context.Context) (callbackParams map[string]any, err error)) (err error) {
	maxCount := -1
	beginRowNumber := ex.startRowNumber
	reqDTO, err := ex.proxy.RequestDTO(requestParams)
	if err != nil {
		return err
	}
	var data []byte // 经过动态脚本格式化原始http body后返回的数据
	loop := 0

	for {
		select {
		case <-ex.context.Done(): // 监听上下文取消
			err = ex.context.Err()
			return err
		default:
		}
		// 开始业务
		loop++
		if loop > MaxLoopTimes {
			err = errors.Errorf("The number of cycles exceeded %d , increase the MaxLoopTimes value or detect whether dynamic scripts incrementally update page info", MaxLoopTimes)
			return err
		}
		requestParams["__loop__"] = loop                                 // 记录循环次数
		reqDTO, data, err = ex.proxy.Request(reqDTO, requestParams, nil) //reqDTO 使用上次格式化的reqDTO,简化动态脚本递增+1翻页
		if err != nil {
			return err
		}
		if data == nil {
			break // 数据为空 跳出循环
		}
		records := make([]map[string]any, 0)
		err = json.Unmarshal(data, &records)
		if err != nil {
			return err
		}
		exchangeData := &excelrw.ExchangeData{
			Data:      make([]map[string]any, 0),
			RowNumber: beginRowNumber,
		}
		exchangeData.Data = records
		beginRowNumber = ex.excelWriteChan.SendData(exchangeData)
		if maxCount < 0 {
			maxCount = len(records)

		} else if maxCount > len(records) {
			break //后面
		}
		if ex.RequestInterval > 0 {
			time.Sleep(ex.RequestInterval) //休眠指定时间
		}
	}
	err = ex.excelWriteChan.Finish()
	if err == nil {
		return err
	}

	var callbackParams map[string]any
	if callback != nil {
		callbackParams, err = callback(ex.context)
		if err != nil {
			return err
		}
	}

	if ex.callbackProxy != nil {
		callbackReqDTO, err := ex.callbackProxy.RequestDTO(callbackParams)
		if err != nil {
			return err
		}
		_, _, err = ex.callbackProxy.Request(callbackReqDTO, callbackParams, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
