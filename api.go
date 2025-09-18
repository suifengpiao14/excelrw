package excelrw

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/suifengpiao14/jsonpathmap"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gitlab.huishoubao.com/gopackage/apihttpprotocol"
	"gitlab.huishoubao.com/gopackage/apihttpprotocol/clientprotocol"
)

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
	Url             string                          `json:"url"  validate:"required"`
	Method          string                          `json:"method" validate:"required"`
	Headers         map[string]string               `json:"headers"`
	Body            json.RawMessage                 `json:"body" validate:"required"`
	PageIndexPath   string                          `json:"pageIndexPath"` //页码参数路径，例如：$.data.pageIndex
	PageSizePath    string                          `json:"pageSizePath"`  //每页大小参数路径，例如：$.data.pageSize
	MiddlewareFuncs apihttpprotocol.MiddlewareFuncs `json:"-"`             // 请求中间件函数列表，一般可以使用动态脚本生成
}
type ProxyResponse struct {
	DataPath        string                          `json:"dataPath"  validate:"required"`
	BusinessOkJson  json.RawMessage                 `json:"businessOkJson"` //业务成功标识json字符串，例如：{"code":0}
	MiddlewareFuncs apihttpprotocol.MiddlewareFuncs `json:"-"`              // 请求中间件函数列表，一般可以使用动态脚本生成
}
type Settings struct {
	Filename        string        `json:"filename" validate:"required"` //导出文件全称如 /static/export/20231018_1547.xlsx
	FieldMetas      FieldMetas    `json:"fieldMetas"`                   //字段映射信息{"id":"ID","name":"姓名"}
	Interval        time.Duration `json:"interval"`
	DeleteFileDelay time.Duration `json:"deleteFileDelay"`
}

type ExportApiIn struct {
	ProxyRquest   ProxyRquest   `json:"proxyRequest" validate:"required"`  //请求数据参数
	ProxyResponse ProxyResponse `json:"proxyResponse" validate:"required"` //响应数据参数
	Settings      Settings      `json:"settings" validate:"required"`      //配置信息
}

const (
	maxLoopTimes = 5000 //最大循环次数，防止死循环
)

// 导出到Excel文件Api ，可直接对接http请求
func ExportApi(in ExportApiIn) (err error) {
	err = validator.New().Struct(in)
	if err != nil {
		return err
	}
	var businessOkkvs jsonpathmap.PathValues
	if len(in.ProxyResponse.BusinessOkJson) > 0 {
		var businessOkJson any
		err = json.Unmarshal(in.ProxyResponse.BusinessOkJson, &businessOkJson)
		if err != nil {
			return err
		}
		businessOkkvs, err = jsonpathmap.FlattenJSON(businessOkJson)
		if err != nil {
			return err
		}
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
	fieldMetas := in.Settings.FieldMetas
	ecw := NewExcelStreamWriter(ctx, filename, fieldMetas)
	ecw = ecw.WithInterval(settings.Interval).WithDeleteFile(deleteFileDelay, nil).WithFetcher(func(loopCount int) (rows []map[string]string, forceBreak bool, err error) {
		if maxLoopTimes < loopCount {
			err = fmt.Errorf("too many loops")
			return nil, false, err
		}

		if proxyReq.PageIndexPath == "" { //不带页码占位符，则只获取一次数据
			forceBreak = true
		}
		body := proxyReq.Body
		pageIndexDelta := loopCount - 1
		if proxyReq.PageSizePath != "" { //带每页大小占位符，则只获取一次数据
			pageIndex := gjson.GetBytes(body, proxyReq.PageIndexPath).Int() + int64(pageIndexDelta)
			body, err = sjson.SetBytes(body, proxyReq.PageSizePath, pageIndex)
			if err != nil {
				return nil, forceBreak, err
			}
		}

		client := clientprotocol.NewRestyClientProtocol(proxyReq.Method, proxyReq.Url)
		client.Request().MiddlewareFuncs.Add(proxyReq.MiddlewareFuncs...)
		client.Response().MiddlewareFuncs.Add(proxyRsp.MiddlewareFuncs...)
		client.Request().SetHeader("Content-Type", "application/json")
		var resp json.RawMessage
		err = client.Do(body, &resp)
		if err != nil {
			return nil, forceBreak, err
		}
		for _, okkv := range businessOkkvs {
			val := gjson.GetBytes(resp, okkv.Path).String()
			expectedVal := cast.ToString(okkv.Value)
			if !strings.EqualFold(val, expectedVal) {
				err = errors.Errorf("business error: expatted:%s,got:%s,response:%s", expectedVal, val, string(resp))
				return nil, forceBreak, err
			}
		}
		data := gjson.GetBytes(resp, proxyRsp.DataPath).Array()
		items := make([]map[string]string, 0)
		if len(fieldMetas) == 0 && len(data) > 0 {
			firstRow := data[0]
			for key := range firstRow.Map() {
				fieldMetas = append(fieldMetas, FieldMeta{Name: key, Title: key})
			}
			ecw = ecw.WithFieldMetas(fieldMetas)
		}
		for _, row := range data {
			rowMap := make(map[string]string)
			for _, fieldMeta := range fieldMetas {
				rowMap[fieldMeta.Name] = row.Get(fieldMeta.Name).String()
				items = append(items, rowMap)
			}
		}
		return items, forceBreak, nil
	})
	errChan, err := ecw.Run()
	if err != nil {
		return err
	}
	err = <-errChan
	if err != nil {
		return err
	}
	return nil
}
