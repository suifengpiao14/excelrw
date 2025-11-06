package repository

import (
	"github.com/suifengpiao14/commonlanguage"
	"github.com/suifengpiao14/sqlbuilder"
)

func NewId(id int) (field *sqlbuilder.Field) {
	return commonlanguage.NewId(id)
}
func NewConfigKey(configKey string) (field *sqlbuilder.Field) {
	return commonlanguage.NewStringId(configKey).SetName("configKey").SetTitle("配置键")
}
func NewUrl(url string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(url, "url", "代理请求地址", 0)
}
func NewMethod(method string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(method, "method", "请求方法", 0)
}

//	func NewUrl(url string) (field *sqlbuilder.Field) {
//		return sqlbuilder.NewStringField(url, "url", "请求地址", 0)
//	}
func NewProxyRequestTpl(proxyRequestTpl string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(proxyRequestTpl, "proxyRequestTpl", "代理获取数据请求模板", 0)
}

// func NewMethod(method string) (field *sqlbuilder.Field) {
// 	return sqlbuilder.NewStringField(method, "method", "请求方法", 0).AppendEnum(
// 		sqlbuilder.Enum{
// 			Key:   http.MethodGet,
// 			Title: http.MethodGet,
// 		},
// 		sqlbuilder.Enum{
// 			Key:   http.MethodPost,
// 			Title: http.MethodPost,
// 		},
// 	)
// }

func NewPageIndexPath(pageIndexPath string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(pageIndexPath, "pageIndexPath", "页码参数路径，例如：data.pageIndex", 0)
}
func NewPageIndexStart(pageIndexStart string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(pageIndexStart, "pageIndexStart", "页码开始值", 0)
}
func NewPageSizePath(pageSizePath string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(pageSizePath, "pageSizePath", "每页数量参数路径(防止前端出入值过小导致循环次数太多)，例如：data.pageSize", 0)
}
func NewPageSize(pageSize string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(pageSize, "pageSize", "每页数量", 0)
}
func NewDataPath(dataPath string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(dataPath, "dataPath", "数据路径，例如：data.list", 0)
}
func NewBusinessCodePath(businessCodePath string) (field *sqlbuilder.Field) {
	return sqlbuilder.NewStringField(businessCodePath, "businessCodePath", "业务成功标识路径，例如：code", 0)
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
func NewCreatedAt(createdAt string) (field *sqlbuilder.Field) {
	return commonlanguage.NewCreatedAt(createdAt)
}
func NewUpdatedAt(updatedAt string) (field *sqlbuilder.Field) {
	return commonlanguage.NewUpdatedAt(updatedAt)
}

func NewDeletedAt() (field *sqlbuilder.Field) {
	return commonlanguage.NewDeletedAt()
}
