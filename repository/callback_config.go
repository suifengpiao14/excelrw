package repository

import (
	"github.com/pkg/errors"
	"github.com/suifengpiao14/httpraw"
	"github.com/suifengpiao14/sqlbuilder"
)

var Export_callback_config_table = sqlbuilder.NewTableConfig("t_export_callback_config").AddColumns(
	sqlbuilder.NewColumn("Fid", sqlbuilder.GetField(NewId)),
	sqlbuilder.NewColumn("Fconfig_key", sqlbuilder.GetField(NewConfigKey)),
	sqlbuilder.NewColumn("Fexport_config_key", sqlbuilder.GetField(NewExportConfigKey)),
	sqlbuilder.NewColumn("Fproxy_request_tpl", sqlbuilder.GetField(NewProxyRequestTpl)),
	sqlbuilder.NewColumn("Fdynamic_script", sqlbuilder.GetField(NewDynamicScript)),
	sqlbuilder.NewColumn("Fbusiness_code_path", sqlbuilder.GetField(NewBusinessCodePath)),
	sqlbuilder.NewColumn("Fbusiness_ok_code", sqlbuilder.GetField(NewBusinessOkCode)),
	sqlbuilder.NewColumn("Fcreated_at", sqlbuilder.GetField(NewCreatedAt)),
	sqlbuilder.NewColumn("Fupdated_at", sqlbuilder.GetField(NewUpdatedAt)),
).AddIndexs(
	sqlbuilder.Index{
		Unique: true,
		ColumnNames: func(table sqlbuilder.TableConfig) (columnNames []string) {
			columnNames = []string{
				table.GetDBNameByFieldNameMust(sqlbuilder.GetFieldName(NewId)),
			}
			return columnNames
		},
	},
)

type ExportCallbackConfig struct {
	Id               int    `gorm:"column:id"  json:"id"`
	ConfigKey        string `gorm:"column:configKey" json:"configKey"`
	ExportConfigKey  string `gorm:"column:exportConfigKey"  json:"exportConfigKey"`
	ProxyRequestTpl  string `gorm:"column:proxyRequestTpl" json:"proxyRequestTpl"`
	DynamicScript    string `gorm:"column:dynamicScript" json:"dynamicScript"`
	BusinessCodePath string `gorm:"column:businessCodePath" json:"businessCodePath"`
	BusinessOkCode   string `gorm:"column:businessOkCode" json:"businessOkCode"`
	CreatedAt        string `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt        string `gorm:"column:updatedAt" json:"updatedAt"`
}

func (m ExportCallbackConfig) RenderRequestDTO(context ...any) (rDTO *httpraw.RequestDTO, err error) {
	rDTO, err = httpraw.RenderRequestDTO(m.ProxyRequestTpl, context...)
	if err != nil {
		err = errors.WithMessagef(err, "ExportCallbackConfig.RenderRequestDTO")
		return nil, err
	}
	return rDTO, nil
}

type ExportCallbackConfigs []ExportCallbackConfig

type ExportCallbackConfigRepository struct {
	table sqlbuilder.TableConfig
}

func NewExportCallbackConfigRepository(tableConfig sqlbuilder.TableConfig) ExportCallbackConfigRepository {
	fieldNames := Export_callback_config_table.Columns.Fields().Names() //从内置表中提取必备字段名
	err := tableConfig.Columns.CheckMissOutFieldName(fieldNames...)     //检测传入表配置中是否缺失内置字段名，如果有则panic退出
	if err != nil {
		panic(err)
	}
	tableConfig = tableConfig.AddIndexs(Export_callback_config_table.Indexs...) //合并索引配置

	s := ExportCallbackConfigRepository{
		table: tableConfig,
	}
	return s
}
func (s ExportCallbackConfigRepository) GetByConfigKey(configKey string) (model ExportCallbackConfig, err error) {
	fs := sqlbuilder.Fields{
		NewConfigKey(configKey).SetRequired(true).AppendWhereFn(sqlbuilder.ValueFnForward).SetDelayApply(func(f *sqlbuilder.Field, fs ...*sqlbuilder.Field) {
			columns := f.GetTable().Columns.DbNameWithAlias().AsAny()
			f.SetSelectColumns(columns...)
		}),
	}
	err = s.table.Repository().FirstMustExists(&model, fs)
	if err != nil {
		return model, err
	}
	return model, nil
}

// func (s ExportCallbackConfigRepository) GetByExportConfigKey(configKeys ...string) (models ExportCallbackConfigs, err error) {
// 	fs := sqlbuilder.Fields{}
// 	for i := range configKeys {
// 		configKey := configKeys[i]
// 		f := NewExportConfigKey(configKey).AppendValueFn(sqlbuilder.ValueFnEmpty2Nil).AppendWhereFn(sqlbuilder.ValueFnFindInSet)
// 		if i == 0 {
// 			f.SetDelayApply(func(f *sqlbuilder.Field, fs ...*sqlbuilder.Field) {
// 				columns := f.GetTable().Columns.DbNameWithAlias().AsAny()
// 				f.SetSelectColumns(columns...)
// 			})
// 		}
// 		fs = fs.Add(f)
// 	}
// 	if len(fs) == 0 {
// 		return nil, errors.Errorf("configKeys is empty")
// 	}
// 	err = s.table.Repository().All(&models, fs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return models, nil
// }
