package repository

import (
	"time"

	"github.com/hoisie/mustache"
	"github.com/suifengpiao14/apihttpprotocol"
	"github.com/suifengpiao14/excelrw/defined"
	"github.com/suifengpiao14/excelrw/dynamichook"
	"github.com/suifengpiao14/sqlbuilder"
	"github.com/suifengpiao14/yaegijson"
)

var Export_config_table = sqlbuilder.NewTableConfig("t_export_config").AddColumns(
	sqlbuilder.NewColumn("config_key", sqlbuilder.GetField(NewConfigKey)),
	sqlbuilder.NewColumn("url", sqlbuilder.GetField(NewUrl)),
	sqlbuilder.NewColumn("method", sqlbuilder.GetField(NewMethod)),
	sqlbuilder.NewColumn("page_index_path", sqlbuilder.GetField(NewPageIndexPath)),
	sqlbuilder.NewColumn("data_path", sqlbuilder.GetField(NewDataPath)),
	sqlbuilder.NewColumn("dynamic_script", sqlbuilder.GetField(NewDynamicScript)),
	sqlbuilder.NewColumn("business_code_path", sqlbuilder.GetField(NewBusinessCodePath)),
	sqlbuilder.NewColumn("business_ok_code", sqlbuilder.GetField(NewBusinessOkCode)),
	sqlbuilder.NewColumn("filename_tpl", sqlbuilder.GetField(NewFilenameTpl)),
	sqlbuilder.NewColumn("field_metas", sqlbuilder.GetField(NewFieldMetas)),
	sqlbuilder.NewColumn("interval", sqlbuilder.GetField(NewInterval)),
	sqlbuilder.NewColumn("delete_file_delay", sqlbuilder.GetField(NewDeleteFileDelay)),
).AddIndexs(
	sqlbuilder.Index{
		Unique: true,
		ColumnNames: func(tableColumns sqlbuilder.ColumnConfigs) (columnNames []string) {
			columnNames = tableColumns.FieldName2ColumnName(
				sqlbuilder.GetFieldName(NewConfigKey),
			)
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

type ExportConfigModel struct {
	ConfigKey        string `json:"configKey"`        //配置键
	Url              string `json:"url"`              //请求地址
	Method           string `json:"method"`           //请求方法，例如：GET,POST
	PageIndexPath    string `json:"pageIndexPath"`    //页码参数路径，例如：$.data.pageIndex``
	DataPath         string `json:"dataPath"`         //数据路径，例如：$.data.list
	BusinessCodePath string `json:"businessCodePath"` //业务成功标识路径，例如：$.code
	BusinessOkCode   string `json:"businessOkCode"`   //业务成功标识值
	FilenameTpl      string `json:"filenameTpl"`      //导出文件全称如 /static/export/{{fielname}}.xlsx
	FieldMetas       string `json:"fieldMetas"`       //字段映射信息，例如：[{"name":"id","title":"title"}]
	Interval         string `json:"interval"`         //间隔时间，例如：10s
	DeleteFileDelay  string `json:"deleteFileDelay"`  //删除文件延迟时间，例如：10s
	DynamicScript    string `json:"dynamicScript"`    //动态脚本
}

func (m ExportConfigModel) ParseFieldMetas() (fieldMetas defined.FieldMetas, err error) {
	fieldMetas = make(defined.FieldMetas, 0)
	err = fieldMetas.Unmarshal(m.FieldMetas)
	if err != nil {
		return nil, err
	}
	return fieldMetas, nil
}

func (m ExportConfigModel) ParseInterval() (interval time.Duration, err error) {
	if m.Interval == "" {
		return 0, nil
	}
	return time.ParseDuration(m.Interval)
}

func (m ExportConfigModel) ParseDeleteFileDelay() (deleteFileDelay time.Duration, err error) {
	if m.DeleteFileDelay == "-1" { // 不删除文件，则延迟时间为0分钟
		return 0, nil
	}
	if m.DeleteFileDelay == "" {
		m.DeleteFileDelay = "10min" //默认延迟10分钟删除文件
	}
	return time.ParseDuration(m.DeleteFileDelay)
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
	filename = mustache.Render(m.FilenameTpl, context...)
	return filename, nil
}

func (m ExportConfigModel) ParseMiddleware() (requestMiddlewareH apihttpprotocol.HandlerFunc[apihttpprotocol.RequestMessage], responseMiddlewareH apihttpprotocol.HandlerFunc[apihttpprotocol.ResponseMessage], err error) {
	dynamicHook := dynamichook.DynamicHook{
		ReqeustMiddlewareName:   "excelrwhook.RequestMiddleware",
		ResponseMiddlewareName:  "excelrwhook.ResponseMiddleware",
		DynamicExtensionHttpRaw: &yaegijson.Extension{},
	}

	requestMiddleware, responseMiddleware, err := dynamicHook.MakeMiddleware()
	if err != nil {
		return nil, nil, err
	}
	if requestMiddleware != nil {
		requestMiddlewareH = requestMiddleware.HandlerFunc()
	}
	if responseMiddleware != nil {
		responseMiddlewareH = responseMiddleware.HandlerFunc()
	}
	return requestMiddlewareH, responseMiddlewareH, nil
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
			f.SetCountColumns(columnsWithAlais...)
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
