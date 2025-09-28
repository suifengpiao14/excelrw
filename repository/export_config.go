package repository

import (
	"time"

	"github.com/hoisie/mustache"
	"github.com/pkg/errors"
	"github.com/suifengpiao14/apihttpprotocol"
	"github.com/suifengpiao14/excelrw/defined"
	"github.com/suifengpiao14/excelrw/dynamichook"
	"github.com/suifengpiao14/sqlbuilder"
	"github.com/suifengpiao14/yaegijson"
)

var IdTimeColumns = sqlbuilder.ColumnConfigs{
	sqlbuilder.NewColumn("Fid", sqlbuilder.GetField(NewId)),
	sqlbuilder.NewColumn("Fcreated_at", sqlbuilder.GetField(NewCreatedAt)),
	sqlbuilder.NewColumn("Fupdated_at", sqlbuilder.GetField(NewUpdatedAt)),
}

var IdIndex = sqlbuilder.Index{
	IsPrimary: true,
	ColumnNames: func(tableColumns sqlbuilder.ColumnConfigs) (columnNames []string) {
		columnNames = tableColumns.FieldName2ColumnName(
			sqlbuilder.GetFieldName(NewId),
		)
		return columnNames
	},
}

var Export_config_table = sqlbuilder.NewTableConfig("t_export_config").AddColumns(
	sqlbuilder.NewColumn("Fconfig_key", sqlbuilder.GetField(NewConfigKey)),
	sqlbuilder.NewColumn("Furl", sqlbuilder.GetField(NewUrl)),
	sqlbuilder.NewColumn("Fmethod", sqlbuilder.GetField(NewMethod)),
	sqlbuilder.NewColumn("Fpage_index_path", sqlbuilder.GetField(NewPageIndexPath)),
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

// ExportConfigModel 导出配置模型结构体，用于解析配置信息,这里gorm:"column:xxx"是固定不变的(查询语句会使用别名转换字段),后续使用 sql.DB，xorm 也可以增加对应的固定tag
type ExportConfigModel struct {
	ConfigKey        string `gorm:"column:configKey" xorm:"'configKey'" db:"configKey" json:"configKey"`                             // 配置键
	Url              string `gorm:"column:url" xorm:"'url'" db:"url" json:"url"`                                                     // 请求地址
	Method           string `gorm:"column:method" xorm:"'method'" db:"method" json:"method"`                                         // 请求方法，例如：GET,POST
	PageIndexPath    string `gorm:"column:pageIndexPath" xorm:"'pageIndexPath'" db:"pageIndexPath" json:"pageIndexPath"`             // 页码参数路径，例如：$.data.pageIndex
	DataPath         string `gorm:"column:dataPath" xorm:"'dataPath'" db:"dataPath" json:"dataPath"`                                 // 数据路径，例如：$.data.list
	BusinessCodePath string `gorm:"column:businessCodePath" xorm:"'businessCodePath'" db:"businessCodePath" json:"businessCodePath"` // 业务成功标识路径，例如：$.code
	BusinessOkCode   string `gorm:"column:businessOkCode" xorm:"'businessOkCode'" db:"businessOkCode" json:"businessOkCode"`         // 业务成功标识值
	FilenameTpl      string `gorm:"column:filenameTpl" xorm:"'filenameTpl'" db:"filenameTpl" json:"filenameTpl"`                     // 导出文件全称如 /static/export/{{fielname}}.xlsx
	FieldMetas       string `gorm:"column:fieldMetas" xorm:"'fieldMetas'" db:"fieldMetas" json:"fieldMetas"`                         // 字段映射信息，例如：[{"name":"id","title":"title"}]
	Interval         string `gorm:"column:interval" xorm:"'interval'" db:"interval" json:"interval"`                                 // 间隔时间，例如：10s
	DeleteFileDelay  string `gorm:"column:deleteFileDelay" xorm:"'deleteFileDelay'" db:"deleteFileDelay" json:"deleteFileDelay"`     // 删除文件延迟时间，例如：10s
	DynamicScript    string `gorm:"column:dynamicScript" xorm:"'dynamicScript'" db:"dynamicScript" json:"dynamicScript"`             // 动态脚本
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
	if m.Interval == Duration_zero { // 不延迟，则间隔时间为0毫秒
		return 0, nil
	}
	if m.Interval == "" {
		m.Interval = "100ms" //默认间隔100毫秒
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
	filename = mustache.Render(m.FilenameTpl, context...)
	return filename, nil
}

func (m ExportConfigModel) ParseMiddleware() (requestMiddleware apihttpprotocol.HandlerFunc[apihttpprotocol.RequestMessage], responseMiddleware apihttpprotocol.HandlerFunc[apihttpprotocol.ResponseMessage], err error) {
	if m.DynamicScript == "" {
		return nil, nil, nil
	}
	dynamicHook := dynamichook.DynamicHook{
		ReqeustMiddlewareName:  "excelrwhook.RequestMiddleware",
		ResponseMiddlewareName: "excelrwhook.ResponseMiddleware",
		DynamicExtensionHttpRaw: &yaegijson.Extension{
			SourceCodes: []string{m.DynamicScript},
		},
	}

	requestMiddleware, responseMiddleware, err = dynamicHook.MakeMiddleware()
	if err != nil {
		return nil, nil, err
	}
	return requestMiddleware, responseMiddleware, nil
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
