package export

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/pkg/errors"

	"github.com/suifengpiao14/excelrw"
	"github.com/suifengpiao14/httpraw"
	"github.com/suifengpiao14/stream"
	"github.com/suifengpiao14/stream/packet"
	"github.com/suifengpiao14/stream/packet/yaegipacket"
)

type SuccessFinishCallbackFn func(ctx context.Context) (err error)
type FailFinishCallbackFn func(ctx context.Context, beforErr error) (err error)

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
	excelWriteChan          *excelrw.ExcelChanWriter
	startRowNumber          int
	Timeout                 time.Duration // 任务处理最长时间
	context                 context.Context
	RequestInterval         time.Duration // 循环请求获取数据的间隔时间
	CancelFunc              context.CancelFunc
	err                     error                   // 异步执行的错误记录
	successFinishCallbackFn SuccessFinishCallbackFn // 成功导出完成后回调函数
	failFinishCallbackFn    FailFinishCallbackFn    // 导出失败后回调函数
	proxyInitStream         stream.PacketHandlers   // 代理请求前处理链路,将template 转换为 http json data
	proxyStream             stream.PacketHandlers   // 代理请求处理链路
	runingProxyHttpJson     *[]byte                 // 记录实际请求时的请求配置数据,用于下次基于此作修改
	callbackStream          stream.PacketHandlers   // 回调处理链路
}

// AsyncError 获取异步错误信息
func (ex *HttpExport) AsyncError() (err error) {
	return ex.err
}

var MaxLoopTimes = 100000 // 最大循环次数，超过这个次数退出循环，并抛出错误

func NewHttpExport(ctx context.Context, exportIn HttpExportIn, option *excelrw.ExcelChanWriterOption, callbackHandlers stream.PacketHandlers) (ex *HttpExport, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ex = &HttpExport{
		Timeout:        exportIn.Timeout,
		context:        ctx,
		callbackStream: callbackHandlers,
	}
	if exportIn.Timeout > 0 {
		timeoutErr := errors.Errorf("proxy export time out")
		ex.context, ex.CancelFunc = context.WithTimeoutCause(ex.context, exportIn.Timeout, timeoutErr)
	}

	sheet := "sheet1"
	ecw, beginRowNumber, err := excelrw.NewExcelChanWriter(ex.context, exportIn.ExcelFilename, sheet, exportIn.FieldMetas, option)
	if err != nil {
		return nil, err
	}
	ex.excelWriteChan = ecw
	ex.startRowNumber = beginRowNumber
	ex.proxyInitStream, ex.proxyStream, err = NewHttprawPacketHandlers(exportIn.Tpl)
	if err != nil {
		return nil, err
	}
	// 增加获取当前请求数据(请求前截取)
	ex.proxyStream.InsertBefore(0, packet.NewFuncPacketHandler(func(ctx context.Context, input []byte) (newCtx context.Context, out []byte, err error) {
		ex.runingProxyHttpJson = &input
		return ctx, input, nil
	}, nil))

	if exportIn.Script != "" {
		yaegiHandler, err := yaegipacket.NewCurlHookYaegi(exportIn.Script) // 修改http json 请求和响应数据
		if err != nil {
			return nil, err
		}
		ex.proxyStream.InsertBefore(0, yaegiHandler) // 插入动态脚本处理流程
	}

	if exportIn.CallbackTpl == "" {
		return ex, nil
	}

	return ex, nil
}

func (ex *HttpExport) AsyncRun(proxyParams string, callbackParams string) {
	go func() {
		defer func() {
			if re := recover(); re != nil {
				ex.err = errors.Errorf("%v+", re)
			}
		}()
		ex.err = ex.Run(proxyParams, callbackParams)
	}()
}

func (ex *HttpExport) SetFinishCallbackFn(successFinishCallbackFn SuccessFinishCallbackFn, failFinishCallbackFn FailFinishCallbackFn) {
	ex.successFinishCallbackFn = successFinishCallbackFn
	ex.failFinishCallbackFn = failFinishCallbackFn
}

var ERROR_FINISHED = errors.New("export finished")

func NewHttprawPacketHandlers(tpl string) (requesHook stream.PacketHandlers, requestHandler stream.PacketHandlers, err error) {
	requesHook = make(stream.PacketHandlers, 0)
	requestHandler = make(stream.PacketHandlers, 0)
	if tpl == "" {
		return
	}

	httpTpl, err := httpraw.NewHttpTpl(tpl)
	if err != nil {
		return nil, nil, err
	}
	//http proxy 请求 前流程处理,将tpl 转为 http json
	tplHandler := packet.NewTemplatePacketHandler(*httpTpl.Template, reflect.TypeOf(make(map[string]any)))
	requesHook.Append(tplHandler)                       //解析模板,输出标准的 http 协议文本
	requesHook.Append(packet.NewHttprawPacketHandler()) // http 协议文本转标准的json格式数据

	// http proxy 请求流程处理,将 http json 经过修改后发器http请求,并格式化http response body 返回

	/*if exportIn.Script != "" {
		yaegiHandler, err := yaegipacket.NewCurlHookYaegi(exportIn.Script) // 修改http json 请求和响应数据
		if err != nil {
			return nil, err
		}
		packHandlers.Append(yaegiHandler)
	}*/
	requestHandler.Append(packet.NewRestyPacketHandler(nil)) // 应用http json 发起curl请求,返回response body
	return requesHook, requestHandler, nil

}

func (ex *HttpExport) Run(proxyParams string, callbackParams string) (err error) {

	s := stream.NewStream(nil, ex.proxyInitStream...)

	httpJson, err := s.Run(ex.context, []byte(proxyParams))
	if err != nil {
		return err
	}

	defer func() {
		if err != nil && ex.failFinishCallbackFn != nil { // 如果失败回调不为空,导出出错后执行结束回调(方便更新数据库状态)
			err = ex.failFinishCallbackFn(ex.context, err)
		}
	}()

	maxCount := -1
	beginRowNumber := ex.startRowNumber
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
		s := stream.NewStream(nil, ex.proxyStream...)
		data, err := s.Run(ex.context, httpJson)
		if err != nil {
			return err
		}
		httpJson = *ex.runingProxyHttpJson // 替换成本次请求的http json 供下次循环使用
		records := make([]map[string]any, 0)
		err = json.Unmarshal(data, &records)
		if err != nil {
			err = errors.WithMessagef(err, "ex.proxyStream response type want:[]map[string]any,got:%s", string(data))
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
	err = ex.successFinishCallbackFn(ex.context)
	if err != nil {
		return err
	}
	if len(ex.callbackStream) < 1 { // 没有回调,直接返回
		return nil
	}
	callbackStream := stream.NewStream(nil, ex.callbackStream...)
	_, err = callbackStream.Run(ex.context, []byte(callbackParams))
	if err != nil {
		return err
	}

	return nil
}
