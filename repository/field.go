package repository

import (
	"net/http"

	"github.com/suifengpiao14/commonlanguage"
	"github.com/suifengpiao14/sqlbuilder"
)

/*
Url             string                                        `json:"url"  validate:"required"`
	Method          string                                        `json:"method" validate:"required"`
	Headers         map[string]string                             `json:"headers"`
	Body            json.RawMessage                               `json:"body" validate:"required"`
	PageIndexPath   string                                        `json:"pageIndexPath"` //页码参数路径，例如：$.data.pageIndex
	MiddlewareFuncs apihttpprotocol.MiddlewareFuncsRequestMessage `json:"-"`             // 请求中间件函数列表，一般可以使用动态脚本生成

	type ProxyResponse struct {
	DataPath         string `json:"dataPath"  validate:"required"`
	BusinessCodePath string `json:"businessCodePath"` //业务成功标识路径，例如：$.code
	BusinessOkCode   string `json:"businessOkCode"`   //业务成功标识值，例如：0
	//	BusinessOkJson   json.RawMessage                                `json:"businessOkJson"`   //业务成功标识json字符串，例如：{"code":0}
	MiddlewareFuncs apihttpprotocol.MiddlewareFuncsResponseMessage `json:"-"` // 请求中间件函数列表，一般可以使用动态脚本生成
}
type Settings struct {
	Filename        string        `json:"filename" validate:"required"` //导出文件全称如 /static/export/20231018_1547.xlsx
	FieldMetas      FieldMetas    `json:"fieldMetas"`                   //字段映射信息{"id":"ID","name":"姓名"}
	Interval        time.Duration `json:"interval"`
	DeleteFileDelay time.Duration `json:"deleteFileDelay"`
}
*/

func NewConfigKey(configKey string) (field *sqlbuilder.Field) {
	return commonlanguage.NewStringId(configKey).SetName("configKey").SetTitle("配置键")
}

func NewUrl(url string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(url, "url", "请求地址", 0)
}
func NewMethod(method string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(method, "method", "请求方法", 0).AppendEnum(
		sqlbuilder.Enum{
			Key:   http.MethodGet,
			Title: http.MethodGet,
		},
		sqlbuilder.Enum{
			Key:   http.MethodPost,
			Title: http.MethodPost,
		},
	)
}

func NewPageIndexPath(pageIndexPath string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(pageIndexPath, "pageIndexPath", "页码参数路径，例如：$.data.pageIndex", 0)
}
func NewDataPath(dataPath string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(dataPath, "dataPath", "数据路径，例如：$.data.list", 0)
}
func NewBusinessCodePath(businessCodePath string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(businessCodePath, "businessCodePath", "业务成功标识路径，例如：$.code", 0)
}
func NewBusinessOkCode(businessOkCode string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(businessOkCode, "businessOkCode", "业务成功标识值，例如：0", 0)
}

func NewFilenameTpl(filenameTpl string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(filenameTpl, "filenameTpl", "导出文件全称如 /static/export/{{fielname}}.xlsx", 0)
}

func NewFieldMetas(fieldMetas string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(fieldMetas, "fieldMetas", `字段映射信息，例如：[{"name":"id","title":"title"}]`, 0)
}

func NewInterval(interval string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(interval, "interval", "间隔时间，例如：10s", 0)
}

func NewDeleteFileDelay(deleteFileDelay string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(deleteFileDelay, "deleteFileDelay", "删除文件延迟时间，例如：10s", 0)
}
func NewDynamicScript(dynamicScript string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(dynamicScript, "dynamicScript", "动态脚本", int(sqlbuilder.Str_Text))
}
