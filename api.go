package excelrw

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/suifengpiao14/apihttpprotocol"
	"github.com/suifengpiao14/excelrw/defined"
	"github.com/suifengpiao14/excelrw/repository"
	"github.com/suifengpiao14/httpraw"
	"github.com/suifengpiao14/sqlbuilder"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var (
//Export_min_page_size int64 = 100 //最小页大小,页码太小循环次数太多，性能不好(该值也不宜设置过大，部分系统列表返回有上限，遇到这种情况，可以将该值赋值为最大上限即可，设置为0，则无最小值限制)

// Export_max_page_size int64 = 100000 //最大页大小,太大内存占用太多，影响稳定性，这个值一般不用修改
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
	//filename := settings.Filename
	// fieldMetas := in.Settings.FieldMetas
	ecw := NewExcelStreamWriter(ctx)
	//startIndex := -1
	//startIndexRaw := ""
	//exp := regexp.MustCompile(`\d+`)
	_rowNumber := 0
	rename := true
	body := proxyReq.Body // body 的pageSize 有可能会被修改，所以在这里赋值
	ecw = ecw.WithInterval(settings.Interval).WithDeleteFile(deleteFileDelay, nil).WithMaxLoopCount(maxLoopTimes).WithFetcher(func(loopTimes int) (rows []map[string]string, forceBreak bool, err error) {
		if proxyReq.RequestFormatFn == nil { //没有定义格式化函数，者无法翻页，直接执行一次请求即退出
			forceBreak = true
		}
		/*
			pageIndexDelta := loopCount - 1

				if proxyReq.PageIndexPath != "" { //带每页大小占位符，则只获取一次数据
					if startIndex == -1 {
						result := gjson.GetBytes(body, proxyReq.PageIndexPath)
						if !result.Exists() {
							err = errors.Errorf("pageIndexPath:%s (not found in body(%s))", proxyReq.PageIndexPath, body)
							return nil, forceBreak, err
						}
						startIndex = int(result.Int())
						if proxyReq.PageIndexStart != "" { // 配置中有，则优先使用配置中的起始值，这样可以避免前端翻页到第二页后点击导出，导致导出数据不全的问题。
							startIndex = cast.ToInt(proxyReq.PageIndexStart)
						}
						startIndexRaw = result.Raw

						if proxyReq.PageSizePath != "" {
							result := gjson.GetBytes(body, proxyReq.PageSizePath)
							if result.Exists() {
								pageSize := result.Int()
								if proxyReq.PageSize > 0 { // 配置中有，则优先使用配置中的每页大小值
									pageSize = cast.ToInt64(proxyReq.PageSize)
								}
								pageSize = max(pageSize, Export_min_page_size)
								pageSize = min(pageSize, Export_max_page_size)
								if pageSize != result.Int() {
									raw := exp.ReplaceAllString(result.Raw, cast.ToString(pageSize))        // 确保类型一致
									body, err = sjson.SetRawBytes(body, proxyReq.PageSizePath, []byte(raw)) //修改body 的pageSize字段值
									if err != nil {
										return nil, forceBreak, err
									}
								}
							}
						}

					}
					pageIndex := startIndex + pageIndexDelta
					indexRaw := exp.ReplaceAllString(startIndexRaw, cast.ToString(pageIndex)) // 确保类型一致
					body, err = sjson.SetRawBytes(body, proxyReq.PageIndexPath, []byte(indexRaw))
					if err != nil {
						return nil, forceBreak, err
					}
				}
		*/
		header := http.Header{}
		for k, v := range in.ProxyRquest.Headers {
			if _, ok := header[k]; !ok { // 这里使用非标准头写入方式，确保大小写和外部传入一致
				header[k] = make([]string, 0)
			}
			header[k] = append(header[k], v)
		}

		requestDTO := httpraw.RequestDTO{
			MetaData: map[string]any{"loopTimes": loopTimes},
			URL:      proxyReq.Url,
			Method:   proxyReq.Method,
			Header:   header,
			Cookies:  make([]*http.Cookie, 0),
			Body:     string(body),
		}

		if in.ProxyRquest.RequestFormatFn != nil {
			oldBody := requestDTO.Body
			newRequestDTO, err := in.ProxyRquest.RequestFormatFn(requestDTO)
			if err != nil {
				return nil, forceBreak, err
			}
			if loopTimes > 1 && strings.EqualFold(oldBody, newRequestDTO.Body) {
				err = errors.Errorf("request body not changed in loop:%d,use loopTimes to increase page index", loopTimes)
				return nil, forceBreak, err
			}
			requestDTO = newRequestDTO
			if titleMap, ok := requestDTO.MetaData["titleMap"]; ok {
				titleMapStr := cast.ToString(titleMap)
				om := orderedmap.New[string, any]() // or orderedmap.New[int, any](), or any type you expect
				err := json.Unmarshal([]byte(titleMapStr), &om)
				if err != nil {
					return nil, forceBreak, err
				}
			}
			if fname, ok := requestDTO.MetaData["filename"]; ok && rename {
				ecw.filename = cast.ToString(fname)
				rename = false // 只重命名一次文件名
			}
		}

		client := apihttpprotocol.NewClientProtocol(requestDTO.Method, requestDTO.URL)
		client.Request().AddMiddleware(proxyReq.MiddlewareFuncs...)
		client.Response().AddMiddleware(proxyRsp.MiddlewareFuncs...)
		client.Request().Headers = requestDTO.Header //设置头

		var resp json.RawMessage
		newBody := json.RawMessage([]byte(requestDTO.Body))
		err = client.Do(newBody, &resp)
		if err != nil {
			return nil, forceBreak, err
		}
		response := client.Response()
		responseDTO := httpraw.ResponseDTO{
			Header:     response.Headers,
			Body:       string(resp),
			RequestDTO: &requestDTO,
		}
		// if in.ProxyResponse.BusinessCodePath != "" {
		// 	businessCode := gjson.GetBytes(resp, in.ProxyResponse.BusinessCodePath).String()
		// 	if businessCode != cast.ToString(in.ProxyResponse.BusinessOkCode) {
		// 		err = errors.Errorf("business error: expatted:%s,got:%s,url:%s,response:%s", in.ProxyResponse.BusinessOkCode, businessCode, proxyReq.Url, string(resp))
		// 		return nil, forceBreak, err
		// 	}
		// }
		var data = make([]map[string]any, 0)
		if in.ProxyResponse.ResponseFormatFn != nil {
			data, err = in.ProxyResponse.ResponseFormatFn(responseDTO)
			if err != nil {
				return nil, forceBreak, err
			}
		}
		items := make([]map[string]string, 0)
		if len(ecw.fieldMetas) == 0 && len(data) > 0 { // 没有传入字段元数据，则自动从第一行获取字段名作为标题
			firstRow := data[0]
			fieldMetas := make([]defined.FieldMeta, 0)
			for key := range firstRow {
				fieldMetas = append(fieldMetas, defined.FieldMeta{Name: key, Title: key})
			}
			ecw = ecw.WithFieldMetas(fieldMetas)
		}

		for _, row := range data {
			_rowNumber++
			rowMap := make(map[string]string)
			rowMap["_rowNumber"] = cast.ToString(_rowNumber)
			for k, v := range row {
				rowMap[k] = cast.ToString(v)
			}

			items = append(items, rowMap)
		}
		return items, forceBreak, nil
	})
	errChan, err = ecw.Run()
	return errChan, err
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
	Url     string            `json:"url"  validate:"required"`
	Method  string            `json:"method" validate:"required"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body" validate:"required"`
	//PageIndexPath   string                                        `json:"pageIndexPath"`  //页码参数路径，例如：$.data.pageIndex
	//PageIndexStart  string                                        `json:"pageIndexStart"` //起始页码，例如："0","1"
	//PageSizePath    string                                        `json:"pageSizePath"`   //每页数量参数路径，例如：$.data.pageSize
	//PageSize        int                                           `json:"pageSize"`       //每页数量，例如：100
	MiddlewareFuncs apihttpprotocol.MiddlewareFuncsRequestMessage `json:"-"` // 请求中间件函数列表，一般可以使用动态脚本生成
	RequestFormatFn defined.RequestFormatFn                       `json:"-"` //请求格式化函数，例如：func(request httpraw.RequestDTO)(newRequest httpraw.RequestDTO,err error){ return request,nil}
}

type ProxyResponse struct {
	//DataPath         string `json:"dataPath"  validate:"required"`
	//BusinessCodePath string `json:"businessCodePath"` //业务成功标识路径，例如：$.code
	//BusinessOkCode   string `json:"businessOkCode"`   //业务成功标识值，例如：0
	//	BusinessOkJson   json.RawMessage                                `json:"businessOkJson"`   //业务成功标识json字符串，例如：{"code":0}
	MiddlewareFuncs  apihttpprotocol.MiddlewareFuncsResponseMessage `json:"-"` // 请求中间件函数列表，一般可以使用动态脚本生成
	ResponseFormatFn defined.ResponseFormatFn                       //格式化记录函数，例如：func(record map[string]string)(newRecord map[string]string,err error){ return record,nil}
}

type Settings struct {
	Filename string `json:"filename" validate:"required"` //导出文件全称如 /static/export/20231018_1547.xlsx
	//FieldMetas      defined.FieldMetas `json:"fieldMetas"`                   //字段映射信息{"id":"ID","name":"姓名"}
	Interval        time.Duration `json:"interval"`
	DeleteFileDelay time.Duration `json:"deleteFileDelay"`
}

type ExportApiIn struct {
	ProxyRquest   ProxyRquest   `json:"proxyRequest" validate:"required"`  //请求数据参数
	ProxyResponse ProxyResponse `json:"proxyResponse" validate:"required"` //响应数据参数
	Settings      Settings      `json:"settings" validate:"required"`      //配置信息
}

const (
	maxLoopTimes = 50000 //最大循环次数，防止死循环
)

type MakeExportApiInArgs struct {
	ConfigKey string   `json:"configKey"` //配置键，例如：user_list
	Request   Request  `json:"request"`   //请求数据参数
	Response  Response `json:"-"`         //响应数据参数,只用于收集中间件,不对外开放
}

type Request struct {
	Body            json.RawMessage                               `json:"body"`    //请求体，例如：{"pageIndex":1}
	Headers         map[string]string                             `json:"headers"` //请求头，例如：{"Content-Type":"application/json"}
	MiddlewareFuncs apihttpprotocol.MiddlewareFuncsRequestMessage `json:"-"`       // 请求中间件函数列表，一般可以使用动态脚本生成
	RequestFormatFn defined.RequestFormatFn                       `json:"-"`       //请求格式化函数，例如：func(request httpraw.RequestDTO)(newRequest httpraw.RequestDTO,err error){ return request,nil}
}

type Response struct {
	MiddlewareFuncs  apihttpprotocol.MiddlewareFuncsResponseMessage `json:"-"` // 请求中间件函数列表，一般可以使用动态脚本生成
	ResponseFormatFn defined.ResponseFormatFn                       `json:"-"` //格式化记录函数，例如：func(record map[string]string)(newRecord map[string]string,err error){ return record,nil}
}

var Export_config_table sqlbuilder.TableConfig = repository.Export_config_table
var IdTimeColumns = repository.IdTimeColumns
var IdIndex = repository.IdIndex

// MakeExportApiIn 生成导出配置信息
func MakeExportApiIn(in MakeExportApiInArgs, table sqlbuilder.TableConfig) (exportApiIn ExportApiIn, err error) {
	exportConfigRepository := repository.NewExportConfigRepository(table)
	getIn := repository.ExportConfigRepositoryGetIn{
		ConfigKey: in.ConfigKey,
	}
	config, err := exportConfigRepository.GetMust(getIn)
	if err != nil {
		return exportApiIn, err
	}
	var requestBody any
	if in.Request.Body != nil {
		err = json.Unmarshal(in.Request.Body, &requestBody)
		if err != nil {
			return exportApiIn, err
		}
	}

	// filename, err := config.ParseFilename(requestBody)
	// if err != nil {
	// 	return exportApiIn, err
	// }
	// fieldMetas, err := config.ParseFieldMetas()
	// if err != nil {
	// 	return exportApiIn, err
	// }
	tnterval, err := config.ParseInterval()
	if err != nil {
		return exportApiIn, err
	}
	deleteFileDelay, err := config.ParseDeleteFileDelay()
	if err != nil {
		return exportApiIn, err
	}

	requestFormatFn, responseFormatFn, err := config.ParseDynamicScript()
	if err != nil {
		return exportApiIn, err
	}
	in.Request.RequestFormatFn = requestFormatFn
	in.Response.ResponseFormatFn = responseFormatFn

	data := map[string]any{
		"body":    requestBody,
		"bodyStr": string(in.Request.Body),
		"headers": in.Request.Headers,
	}
	reqDTO, err := config.ParseRequest(data)
	if err != nil {
		return exportApiIn, err
	}

	header := make(map[string]string)
	for k, vArr := range reqDTO.Header {
		for _, v := range vArr {
			header[k] = v
			break
		}
	}
	maps.Copy(header, in.Request.Headers)
	exportApiIn = ExportApiIn{
		ProxyRquest: ProxyRquest{
			Url:    reqDTO.URL,
			Method: reqDTO.Method,
			// PageIndexPath:   config.PageIndexPath,
			// PageIndexStart:  config.PageIndexStart,
			// PageSizePath:    config.PageSizePath,
			// PageSize:        config.PageSize,
			Body:            json.RawMessage(reqDTO.Body),
			Headers:         header,
			MiddlewareFuncs: in.Request.MiddlewareFuncs,
			RequestFormatFn: in.Request.RequestFormatFn,
		}, //请求数据参数
		ProxyResponse: ProxyResponse{
			// DataPath:         config.DataPath,
			// BusinessCodePath: config.BusinessCodePath,
			// BusinessOkCode:   config.BusinessOkCode,
			MiddlewareFuncs:  in.Response.MiddlewareFuncs,
			ResponseFormatFn: in.Response.ResponseFormatFn,
		}, //响应数据参数
		Settings: Settings{
			//Filename: filename,
			// FieldMetas:      fieldMetas,
			Interval:        tnterval,
			DeleteFileDelay: deleteFileDelay,
		}, //配置信息
	}
	return exportApiIn, nil
}
