package repository

import (
	"time"

	"github.com/cbroglie/mustache"
	"github.com/pkg/errors"
	"github.com/suifengpiao14/excelrw/defined"
	"github.com/suifengpiao14/excelrw/dynamichook"
	"github.com/suifengpiao14/httpraw"
	"github.com/suifengpiao14/sqlbuilder"
)

var IdTimeColumns = sqlbuilder.ColumnConfigs{
	sqlbuilder.NewColumn("Fid", sqlbuilder.GetField(NewId)),
	sqlbuilder.NewColumn("Fcreated_at", sqlbuilder.GetField(NewCreatedAt)),
	sqlbuilder.NewColumn("Fupdated_at", sqlbuilder.GetField(NewUpdatedAt)),
}

var IdIndex = sqlbuilder.Index{
	IsPrimary: true,
	ColumnNames: func(table sqlbuilder.TableConfig) (columnNames []string) {
		columnNames = []string{
			table.GetDBNameByFieldNameMust(sqlbuilder.GetFieldName(NewId)),
		}
		return columnNames
	},
}

var Export_config_table = sqlbuilder.NewTableConfig("t_export_config").AddColumns(
	sqlbuilder.NewColumn("Fconfig_key", sqlbuilder.GetField(NewConfigKey)),
	sqlbuilder.NewColumn("Fproxy_request_tpl", sqlbuilder.GetField(NewProxyRequestTpl)),
	sqlbuilder.NewColumn("Fpage_index_path", sqlbuilder.GetField(NewPageIndexPath)),
	sqlbuilder.NewColumn("Fpage_index_start", sqlbuilder.GetField(NewPageIndexStart)),
	sqlbuilder.NewColumn("Fpage_size_path", sqlbuilder.GetField(NewPageSizePath)),
	sqlbuilder.NewColumn("Fpage_size", sqlbuilder.GetField(NewPageSize)),
	sqlbuilder.NewColumn("Fdata_path", sqlbuilder.GetField(NewDataPath)),
	sqlbuilder.NewColumn("Fdynamic_script", sqlbuilder.GetField(NewDynamicScript)),
	sqlbuilder.NewColumn("Fbusiness_code_path", sqlbuilder.GetField(NewBusinessCodePath)),
	sqlbuilder.NewColumn("Fbusiness_ok_code", sqlbuilder.GetField(NewBusinessOkCode)),
	sqlbuilder.NewColumn("Ffilename_tpl", sqlbuilder.GetField(NewFilenameTpl)),
	sqlbuilder.NewColumn("Ffield_metas", sqlbuilder.GetField(NewFieldMetas)),
	sqlbuilder.NewColumn("Finterval", sqlbuilder.GetField(NewInterval)),
	sqlbuilder.NewColumn("Fdelete_file_delay", sqlbuilder.GetField(NewDeleteFileDelay)),
).AddIndexs(
	sqlbuilder.Index{
		Unique: true,
		ColumnNames: func(table sqlbuilder.TableConfig) (columnNames []string) {
			columnNames = []string{
				table.GetDBNameByFieldNameMust(sqlbuilder.GetFieldName(NewConfigKey)),
			}
			return columnNames
		},
	},
)

type ExportConfigRepository struct {
	table sqlbuilder.TableConfig
}

func NewExportConfigRepository(tableConfig sqlbuilder.TableConfig) ExportConfigRepository {
	fieldNames := Export_config_table.Columns.Fields().Names()      //从内置表中提取必备字段名
	err := tableConfig.Columns.CheckMissOutFieldName(fieldNames...) //检测传入表配置中是否缺失内置字段名，如果有则panic退出
	if err != nil {
		panic(err)
	}
	tableConfig = tableConfig.AddIndexs(Export_config_table.Indexs...) //合并索引配置

	s := ExportConfigRepository{
		table: tableConfig,
	}
	return s
}

// ExportConfigModel 导出配置模型结构体，用于解析配置信息,这里gorm:"column:xxx"是固定不变的(查询语句会使用别名转换字段),后续使用 sql.DB，xorm 也可以增加对应的固定tag
type ExportConfigModel struct {
	ConfigKey         string `gorm:"column:configKey" xorm:"'configKey'" db:"configKey" json:"configKey"`                                 // 配置键
	ProxyRequestTpl   string `gorm:"column:proxyRequestTpl" xorm:"'proxyRequestTpl'" db:"proxyRequestTpl" json:"proxyRequestTpl"`         // 代理获取数据请求模板，例如：{{.Url}}?pageIndex={{.PageIndex}}
	ReqeustPagination string `gorm:"column:reqeustPagination" xorm:"'reqeustPagination'" db:"reqeustPagination" json:"reqeustPagination"` // 请求分页参数，例如：pageIndex,pageSize
	PageIndexPath     string `gorm:"column:pageIndexPath" xorm:"'pageIndexPath'" db:"pageIndexPath" json:"pageIndexPath"`                 // 页码参数路径，例如：$.data.pageIndex
	PageIndexStart    string `gorm:"column:pageIndexStart" xorm:"'pageIndexStart'" db:"pageIndexStart" json:"pageIndexStart"`             // 页码起始值，例如：1
	PageSizePath      string `gorm:"column:pageSizePath" xorm:"'pageSizePath'" db:"pageSizePath" json:"pageSizePath"`                     // 每页数量参数路径，例如：$.data.pageSize
	PageSize          int    `gorm:"column:pageSize" xorm:"'pageSize'" db:"pageSize" json:"pageSize"`                                     // 每页数量，例如：10
	DataPath          string `gorm:"column:dataPath" xorm:"'dataPath'" db:"dataPath" json:"dataPath"`                                     // 数据路径，例如：$.data.list
	BusinessCodePath  string `gorm:"column:businessCodePath" xorm:"'businessCodePath'" db:"businessCodePath" json:"businessCodePath"`     // 业务成功标识路径，例如：$.code
	BusinessOkCode    string `gorm:"column:businessOkCode" xorm:"'businessOkCode'" db:"businessOkCode" json:"businessOkCode"`             // 业务成功标识值
	FilenameTpl       string `gorm:"column:filenameTpl" xorm:"'filenameTpl'" db:"filenameTpl" json:"filenameTpl"`                         // 导出文件全称如 /static/export/{{fielname}}.xlsx
	FieldMetas        string `gorm:"column:fieldMetas" xorm:"'fieldMetas'" db:"fieldMetas" json:"fieldMetas"`                             // 字段映射信息，例如：[{"name":"id","title":"title"}]
	Interval          string `gorm:"column:interval" xorm:"'interval'" db:"interval" json:"interval"`                                     // 间隔时间，例如：10s
	DeleteFileDelay   string `gorm:"column:deleteFileDelay" xorm:"'deleteFileDelay'" db:"deleteFileDelay" json:"deleteFileDelay"`         // 删除文件延迟时间，例如：10s
	DynamicScript     string `gorm:"column:dynamicScript" xorm:"'dynamicScript'" db:"dynamicScript" json:"dynamicScript"`                 // 动态脚本
}

func (m ExportConfigModel) ParseFieldMetas() (fieldMetas defined.FieldMetas, err error) {
	fieldMetas = make(defined.FieldMetas, 0)
	if m.FieldMetas == "" {
		return fieldMetas, nil
	}
	err = fieldMetas.Unmarshal(m.FieldMetas)
	if err != nil {
		err = errors.WithMessagef(err, "json string:%s", m.FieldMetas)
		return nil, err
	}
	return fieldMetas, nil
}

const (
	Duration_zero = "-1"
)

func (m ExportConfigModel) ParseInterval() (interval time.Duration, err error) {
	if m.Interval == "" {
		return 0, nil
	}
	interval, err = time.ParseDuration(m.Interval)
	if err != nil {
		err = errors.WithMessagef(err, "time.ParseDuration(%s)", m.Interval)
		return 0, err
	}
	return interval, nil
}

func (m ExportConfigModel) ParseDeleteFileDelay() (deleteFileDelay time.Duration, err error) {
	if m.DeleteFileDelay == Duration_zero { // 不删除文件，则延迟时间为0分钟
		return 0, nil
	}
	if m.DeleteFileDelay == "" {
		m.DeleteFileDelay = "10m" //默认延迟10分钟删除文件
	}
	deleteFileDelay, err = time.ParseDuration(m.DeleteFileDelay)
	if err != nil {
		err = errors.WithMessagef(err, "time.ParseDuration(%s)", m.DeleteFileDelay)
		return 0, err
	}

	return deleteFileDelay, nil
}

func (m ExportConfigModel) ParseFilename(context ...any) (filename string, err error) {
	if m.FilenameTpl == "" {
		return "", nil
	}
	if context == nil {
		context = make([]any, 0)
	}
	data := make(map[string]any)

	data["datetime"] = time.Now().Local().Format("20060102150405")
	context = append(context, data)
	filename, err = mustache.Render(m.FilenameTpl, context...)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// func (m ExportConfigModel) ParseDynamicMiddleware() (out dynamichook.DynamicMiddleware, err error) {
// 	if m.DynamicScript == "" {
// 		return out, nil
// 	}
// 	dynamicHook := dynamichook.DynamicHook{
// 		ReqeustMiddlewareName:  "excelrwhook.RequestMiddleware",
// 		ResponseMiddlewareName: "excelrwhook.ResponseMiddleware",
// 		DynamicExtension: &yaegijson.Extension{
// 			SourceCodes: []string{m.DynamicScript},
// 		},
// 	}

// 	out, err = dynamicHook.MakeMiddleware()
// 	if err != nil {
// 		return out, err
// 	}
// 	return out, nil
// }

var RecordFormatFnName = "recordFormatFn"
var RequestFormatFnName = "requestFormatFn"
var ResponseFormatFnName = "responseFormatFn"
var SettingFnName = "settingFn"

type DynamicFn struct {
	SettingFn        defined.SettingFn
	RequestFormatFn  defined.RequestFormatFn
	ResponseFormatFn defined.ResponseFormatFn
	RecordFormatFn   defined.RecordFormatFn
}

func (m ExportConfigModel) ParseDynamicScript() (dynamicFn DynamicFn, err error) {
	if m.DynamicScript == "" {
		return dynamicFn, nil
	}
	jsvm, err := dynamichook.ParseJSVM(m.DynamicScript)
	if err != nil {
		return dynamicFn, err
	}

	recordFormatFn, err := jsvm.RecordFormatFn(RecordFormatFnName)
	if err != nil {
		if errors.Is(err, dynamichook.ErrorJSNotFound) {
			err = nil
		}
	}
	if err != nil {
		return dynamicFn, err
	}
	dynamicFn.RecordFormatFn = recordFormatFn

	requestFormatFn, err := jsvm.RequestFormatFn(RequestFormatFnName)
	if err != nil {
		if errors.Is(err, dynamichook.ErrorJSNotFound) {
			err = nil
		}
	}
	if err != nil {
		return dynamicFn, err
	}
	dynamicFn.RequestFormatFn = requestFormatFn

	responseFormatFn, err := jsvm.ResponseFormatFn(ResponseFormatFnName)
	if err != nil {
		if errors.Is(err, dynamichook.ErrorJSNotFound) {
			err = nil
		}
	}
	if err != nil {
		return dynamicFn, err
	}
	dynamicFn.ResponseFormatFn = responseFormatFn

	settingFn, err := jsvm.SettingFn(SettingFnName)
	if err != nil {
		if errors.Is(err, dynamichook.ErrorJSNotFound) {
			err = nil
		}
	}
	if err != nil {
		return dynamicFn, err
	}

	dynamicFn.SettingFn = settingFn

	return dynamicFn, nil
}

func (m ExportConfigModel) RenderRequestDTO(context ...any) (rDTO *httpraw.RequestDTO, err error) {
	rDTO, err = httpraw.RenderRequestDTO(m.ProxyRequestTpl, context...)
	if err != nil {
		err = errors.WithMessagef(err, "ExportConfigModel.RenderRequestDTO")
		return nil, err
	}
	return rDTO, nil
}

type ExportConfigModels []ExportConfigModel

type ExportConfigRepositoryGetIn struct {
	ConfigKey   string
	ExtraFields sqlbuilder.Fields
}

func (in ExportConfigRepositoryGetIn) Fields() sqlbuilder.Fields {
	fs := sqlbuilder.Fields{
		NewConfigKey(in.ConfigKey).SetRequired(true).AppendWhereFn(sqlbuilder.ValueFnForward).SetDelayApply(func(f *sqlbuilder.Field, fs ...*sqlbuilder.Field) {
			fieldNames := Export_config_table.Columns.Fields().Names()
			columns := f.GetTable().Columns.FilterByFieldName(fieldNames...)
			columnsWithAlais := columns.DbNameWithAlias().AsAny()
			f.SetSelectColumns(columnsWithAlais...)
		}),
	}
	fs = fs.Add(in.ExtraFields...)
	return fs
}

func (s ExportConfigRepository) GetMust(in ExportConfigRepositoryGetIn) (model *ExportConfigModel, err error) {
	model = &ExportConfigModel{}
	err = s.table.Repository().FirstMustExists(model, in.Fields())
	if err != nil {
		return nil, err
	}
	return model, nil
}
