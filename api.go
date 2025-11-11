package excelrw

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"regexp"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/suifengpiao14/apihttpprotocol"
	"github.com/suifengpiao14/excelrw/defined"
	"github.com/suifengpiao14/excelrw/repository"
	"github.com/suifengpiao14/httpraw"
	"github.com/suifengpiao14/sqlbuilder"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	Export_min_page_size int64 = 100 //最小页大小,页码太小循环次数太多，性能不好(该值也不宜设置过大，部分系统列表返回有上限，遇到这种情况，可以将该值赋值为最大上限即可，设置为0，则无最小值限制)

	Export_max_page_size int64 = 100000 //最大页大小,太大内存占用太多，影响稳定性，这个值一般不用修改
)

var MessageLogger = watermill.NewStdLogger(false, false)

//var exportTopic = "export_topic_463b0a36567f01d8de4ac691aa4167da"

// type ExportEvent struct {
// 	EventID string `json:"eventId"`
// 	FileUrl string `json:"fileUrl"`
// }

// func (event ExportEvent) Publish() (err error) {
// 	msg, err := event.toMessage()
// 	if err != nil {
// 		return err
// 	}
// 	err = domaineventpubsub.Publish(exportTopic, event.EventID, msg)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func (event ExportEvent) toMessage() (msg *message.Message, err error) {
// 	b, err := json.Marshal(event)
// 	if err != nil {
// 		return nil, err
// 	}
// 	msg = message.NewMessage(watermill.NewUUID(), b)
// 	return msg, nil
// }

// func RegisterCallback(callBackFns ...CallBackFnV2) (err error) {
// 	consumers := make([]domaineventpubsub.Consumer, 0)
// 	for _, callBackFn := range callBackFns {
// 		if callBackFn == nil {
// 			continue
// 		}
// 		workFn := func(event ExportEvent) error {
// 			return callBackFn(event.FileUrl)
// 		}
// 		consumers = append(consumers, domaineventpubsub.Consumer{
// 			Description: "导出任务完成事件",
// 			Topic:       exportTopic,
// 			RouteKey:    ExportEvent_EventID_finished,
// 			WorkFn:      domaineventpubsub.MakeWorkFn(workFn),
// 		})
// 	}

// 	err = domaineventpubsub.RegisterConsumer(consumers...)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

const (
	ExportEvent_EventID_finished = "finished" //导出完成事件ID
)

// 导出到Excel文件Api ，可直接对接http请求
func ExportApi(in ExportApiIn) (errChan chan error, err error) {
	err = validator.New().Struct(in)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	settings := in.Settings
	if settings.Interval == 0 {
		settings.Interval = 100 * time.Millisecond
	}
	deleteFileDelay := settings.DeleteFileDelay
	if deleteFileDelay == 0 {
		deleteFileDelay = 10 * time.Minute
	}
	proxyReq := in.ProxyRquest
	proxyRsp := in.ProxyResponse
	filename := settings.Filename
	ecw := NewExcelStreamWriter(ctx, filename).WithFieldMetas(in.Settings.FieldMetas)
	startIndex := 0
	startIndexRaw := ""
	exp := regexp.MustCompile(`\d+`)
	_rowNumber := 0
	bodyDefault := []byte(proxyReq.Body) // body 的pageSize 有可能会被修改，所以在这里赋值

	if proxyReq.PageIndexPath != "" {
		//获取pageIndex文本,在循环内替换
		result := gjson.GetBytes(bodyDefault, proxyReq.PageIndexPath)
		if !result.Exists() {
			err = errors.Errorf("pageIndexPath:%s (not found in body(%s))", proxyReq.PageIndexPath, bodyDefault)
			return nil, err
		}
		startIndex = int(result.Int())
		if proxyReq.PageIndexStart != "" { // 配置中有，则优先使用配置中的起始值，这样可以避免前端翻页到第二页后点击导出，导致导出数据不全的问题。
			startIndex = cast.ToInt(proxyReq.PageIndexStart)
		}
		startIndexRaw = result.Raw
		//修正pageSize值
		if proxyReq.PageSizePath != "" {
			result := gjson.GetBytes(bodyDefault, proxyReq.PageSizePath)
			if result.Exists() {
				pageSize := result.Int()
				if proxyReq.PageSize > 0 { // 配置中有，则优先使用配置中的每页大小值
					pageSize = cast.ToInt64(proxyReq.PageSize)
				}
				pageSize = max(pageSize, Export_min_page_size)
				pageSize = min(pageSize, Export_max_page_size)
				if pageSize != result.Int() {
					raw := exp.ReplaceAllString(result.Raw, cast.ToString(pageSize))                      // 确保类型一致
					bodyDefault, err = sjson.SetRawBytes(bodyDefault, proxyReq.PageSizePath, []byte(raw)) //修改body 的pageSize字段值
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}
	maxLoopTimes := MaxLoopTimes
	if proxyReq.PageIndexPath == "" { //不带页码占位符，则只获取一次数据
		maxLoopTimes = 1 // 只获取一次数据
	}

	requestDTODefault := httpraw.RequestDTO{
		URL:     proxyReq.Url,
		Method:  proxyReq.Method,
		Headers: in.ProxyRquest.Headers,
		Cookies: make([]*http.Cookie, 0),
		Body:    string(bodyDefault),
	}

	ecw = ecw.WithInterval(settings.Interval).WithDeleteFile(deleteFileDelay, nil).WithMaxLoopCount(maxLoopTimes).WithFetcher(func(loopTimes int) (rows []map[string]string, err error) {
		pageIndexDelta := loopTimes - 1
		requestDTO := requestDTODefault
		if proxyReq.PageIndexPath != "" {
			pageIndex := startIndex + pageIndexDelta
			indexRaw := exp.ReplaceAllString(startIndexRaw, cast.ToString(pageIndex)) // 确保类型一致
			requestDTO.Body, err = sjson.SetRaw(requestDTO.Body, proxyReq.PageIndexPath, indexRaw)
			if err != nil {
				return nil, err
			}
		}

		if in.ProxyRquest.RequestFormatFn != nil {
			newRequestDTO, err := in.ProxyRquest.RequestFormatFn(requestDTO)
			if err != nil {
				return nil, err
			}
			requestDTO = newRequestDTO
		}

		client := apihttpprotocol.NewClientProtocol(requestDTO.Method, requestDTO.URL)
		client.Request().AddMiddleware(proxyReq.MiddlewareFuncs...)
		client.Response().AddMiddleware(proxyRsp.MiddlewareFuncs...)
		client.Request().Headers = requestDTO.Headers.HttpHeaders() //设置头

		var resp json.RawMessage
		newBody := json.RawMessage([]byte(requestDTO.Body))
		err = client.Do(newBody, &resp)
		if err != nil {
			err = errors.WithMessagef(err, "curl:%s", client.Request().CurlCommand())
			return nil, err
		}
		if in.ProxyResponse.BusinessCodePath != "" {
			businessCode := gjson.GetBytes(resp, in.ProxyResponse.BusinessCodePath).String()
			if businessCode != cast.ToString(in.ProxyResponse.BusinessOkCode) {
				err = ProxyResponseError{
					ExpattedBusinessCode: in.ProxyResponse.BusinessOkCode,
					ActualBusinessCode:   businessCode,
					Url:                  proxyReq.Url,
					Response:             string(resp),
				}
				return nil, err
			}
		}
		data := gjson.GetBytes(resp, proxyRsp.DataPath).Array()

		if len(ecw.fieldMetas) == 0 && len(data) > 0 { // 没有传入字段元数据，则自动从第一行获取字段名作为标题
			firstRow := data[0]
			fieldMetas := make([]defined.FieldMeta, 0)
			for key := range firstRow.Map() {
				fieldMetas = append(fieldMetas, defined.FieldMeta{Name: key, Title: key})
			}
			ecw = ecw.WithFieldMetas(fieldMetas)
		}

		items := make([]map[string]string, 0)
		for _, row := range data {
			_rowNumber++
			rowMap := make(map[string]string)
			rowMap["_rowNumber"] = cast.ToString(_rowNumber)
			for k, v := range row.Map() {
				rowMap[k] = v.String()
			}
			if in.ProxyResponse.RecordFormatFn != nil {
				rowMap, err = in.ProxyResponse.RecordFormatFn(rowMap)
				if err != nil {
					return nil, err
				}
			}
			items = append(items, rowMap)
		}
		return items, nil
	})
	errChan, err = ecw.Run()
	return errChan, err
}

type ProxyResponseError struct {
	ExpattedBusinessCode string `json:"expattedBusinessCode"`
	ActualBusinessCode   string `json:"actualBusinessCode"`
	Url                  string `json:"url"`
	Response             string `json:"response"`
}

func (p ProxyResponseError) Error() string {
	var w bytes.Buffer
	encoder := json.NewEncoder(&w)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(p)
	if err != nil {
		return err.Error()
	}
	return w.String()
}

type SimpleExportOut struct {
	Url string `json:"url"`
}

/*
|request|object | 是  | 无 | 代理请求参数|
|request.url|string | 是  | 无 | 代理请求地址 |
|request.method|string | 是  | 无 | 代理请求方法 |
|request.headers|string | 是  | 无 | 代理请求头 |
|request.body|string | 是  | 无 | 代理请求body 体（页码参数必须使用{{pageIndex}}占位符） |
*/

type ProxyRquest struct {
	Url             string                                        `json:"url"  validate:"required"`
	Method          string                                        `json:"method" validate:"required"`
	Headers         map[string]string                             `json:"headers"`
	Body            json.RawMessage                               `json:"body" validate:"required"`
	PageIndexPath   string                                        `json:"pageIndexPath"`  //页码参数路径，例如：$.data.pageIndex
	PageIndexStart  string                                        `json:"pageIndexStart"` //起始页码，例如："0","1"
	PageSizePath    string                                        `json:"pageSizePath"`   //每页数量参数路径，例如：$.data.pageSize
	PageSize        int                                           `json:"pageSize"`       //每页数量，例如：100
	MiddlewareFuncs apihttpprotocol.MiddlewareFuncsRequestMessage `json:"-"`              // 请求中间件函数列表，一般可以使用动态脚本生成
	RequestFormatFn defined.RequestFormatFn                       `json:"-"`              //请求格式化函数，例如：func(request httpraw.RequestDTO)(newRequest httpraw.RequestDTO,err error){ return request,nil}
}

type ProxyResponse struct {
	DataPath         string                                         `json:"dataPath"  validate:"required"`
	BusinessCodePath string                                         `json:"businessCodePath"` //业务成功标识路径，例如：$.code
	BusinessOkCode   string                                         `json:"businessOkCode"`   //业务成功标识值，例如：0
	MiddlewareFuncs  apihttpprotocol.MiddlewareFuncsResponseMessage `json:"-"`                // 请求中间件函数列表，一般可以使用动态脚本生成
	RecordFormatFn   defined.RecordFormatFn                         //格式化记录函数，例如：func(record map[string]string)(newRecord map[string]string,err error){ return record,nil}
}

type Settings struct {
	Filename        string             `json:"filename" validate:"required"` //导出文件全称如 /static/export/20231018_1547.xlsx
	FieldMetas      defined.FieldMetas `json:"fieldMetas"`                   //字段映射信息{"id":"ID","name":"姓名"}
	Interval        time.Duration      `json:"interval"`
	DeleteFileDelay time.Duration      `json:"deleteFileDelay"`
}

type ExportApiIn struct {
	ProxyRquest   ProxyRquest   `json:"proxyRequest" validate:"required"`  //请求数据参数
	ProxyResponse ProxyResponse `json:"proxyResponse" validate:"required"` //响应数据参数
	Settings      Settings      `json:"settings" validate:"required"`      //配置信息
	//CallBackFns   []CallBackFnV2 `json:"-"`                                 //回调函数列表，例如：func(fileUrl string)(err error){ return nil}
}

const (
	MaxLoopTimes = 50000 //最大循环次数，防止死循环
)

type MakeExportApiInArgs struct {
	Async     bool     `json:"async"`     //是否异步执行，默认同步
	CreatorId string   `json:"creatorId"` //创建者ID，例如：1
	ConfigKey string   `json:"configKey"` //配置键，例如：user_list
	Request   Request  `json:"request"`   //请求数据参数
	response  Response `json:"-"`         //响应数据参数,只用于收集中间件,不对外开放
}

type Request struct {
	Body            json.RawMessage                               `json:"body"`    //请求体，例如：{"pageIndex":1}
	Headers         map[string]string                             `json:"headers"` //请求头，例如：{"Content-Type":"application/json"}
	MiddlewareFuncs apihttpprotocol.MiddlewareFuncsRequestMessage `json:"-"`       // 请求中间件函数列表，一般可以使用动态脚本生成
	RequestFormatFn defined.RequestFormatFn                       `json:"-"`       //请求格式化函数，例如：func(request httpraw.RequestDTO)(newRequest httpraw.RequestDTO,err error){ return request,nil}
}

type Response struct {
	MiddlewareFuncs apihttpprotocol.MiddlewareFuncsResponseMessage `json:"-"` // 请求中间件函数列表，一般可以使用动态脚本生成
	RecordFormatFn  defined.RecordFormatFn                         `json:"-"` //格式化记录函数，例如：func(record map[string]string)(newRecord map[string]string,err error){ return record,nil}
}

var Export_config_table sqlbuilder.TableConfig = repository.Export_config_table
var IdTimeColumns = repository.IdTimeColumns
var IdIndex = repository.IdIndex

type TableConfig struct {
	ConfigTable         sqlbuilder.TableConfig
	ConfigCallbackTable sqlbuilder.TableConfig
	TaskTable           sqlbuilder.TableConfig
	CallbacTaskTable    sqlbuilder.TableConfig
}

func init() {
	httpraw.NonstandardHeaderKeyMap = map[string]string{
		"Hsb-Openapi-Callerserviceid": "HSB-OPENAPI-CALLERSERVICEID",
		"Hsb-Openapi-Signature":       "HSB-OPENAPI-SIGNATURE",
	}
}

// MakeExportApiIn 生成导出配置信息
func MakeExportApiIn(in MakeExportApiInArgs, config repository.ExportConfigModel) (exportApiIn ExportApiIn, err error) {
	var requestBody any
	if in.Request.Body != nil {
		err = json.Unmarshal(in.Request.Body, &requestBody)
		if err != nil {
			return exportApiIn, err
		}
	}

	data := map[string]any{
		"creatorId": in.CreatorId,
		"body":      string(in.Request.Body),
	}

	filename, err := config.ParseFilename(data, requestBody)
	if err != nil {
		return exportApiIn, err
	}
	fieldMetas, err := config.ParseFieldMetas()
	if err != nil {
		return exportApiIn, err
	}
	tnterval, err := config.ParseInterval()
	if err != nil {
		return exportApiIn, err
	}
	deleteFileDelay, err := config.ParseDeleteFileDelay()
	if err != nil {
		return exportApiIn, err
	}

	dynamicFn, err := config.ParseDynamicScript()
	if err != nil {
		return exportApiIn, err
	}
	in.Request.RequestFormatFn = dynamicFn.RequestFormatFn
	in.response.RecordFormatFn = dynamicFn.RecordFormatFn

	reqDTO, err := config.RenderRequestDTO(data, requestBody)
	if err != nil {
		return exportApiIn, err
	}
	header := reqDTO.Headers
	maps.Copy(header, in.Request.Headers)
	exportApiIn = ExportApiIn{
		ProxyRquest: ProxyRquest{
			Url:             reqDTO.URL,
			Method:          reqDTO.Method,
			PageIndexPath:   config.PageIndexPath,
			PageIndexStart:  config.PageIndexStart,
			PageSizePath:    config.PageSizePath,
			PageSize:        config.PageSize,
			Body:            json.RawMessage(reqDTO.Body),
			Headers:         header,
			MiddlewareFuncs: in.Request.MiddlewareFuncs,
			RequestFormatFn: in.Request.RequestFormatFn,
		}, //请求数据参数
		ProxyResponse: ProxyResponse{
			DataPath:         config.DataPath,
			BusinessCodePath: config.BusinessCodePath,
			BusinessOkCode:   config.BusinessOkCode,
			MiddlewareFuncs:  in.response.MiddlewareFuncs,
			RecordFormatFn:   in.response.RecordFormatFn,
		}, //响应数据参数
		Settings: Settings{
			Filename:        filename,
			FieldMetas:      fieldMetas,
			Interval:        tnterval,
			DeleteFileDelay: deleteFileDelay,
		}, //配置信息
		// CallBackFns: []CallBackFnV2{
		// 	func(fileUrl string) (err error) {
		// 		event := ExportEvent{
		// 			EventID: ExportEvent_EventID_finished,
		// 			FileUrl: fileUrl,
		// 		}
		// 		err = event.Publish()
		// 		if err != nil {
		// 			return err
		// 		}
		// 		return nil
		// 	},
		// },
	}
	return exportApiIn, nil
}
