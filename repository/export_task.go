package repository

import (
	"github.com/spf13/cast"
	"github.com/suifengpiao14/sqlbuilder"
)

/*
CREATE TABLE `export_task` (

	`id` bigint(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增ID',
	`app_id` varchar(128) NOT NULL DEFAULT '' COMMENT 'APP标识',
	`name` varchar(64) NOT NULL DEFAULT '' COMMENT '任务名称',
	`creator_id` varchar(64) NOT NULL DEFAULT '' COMMENT '创建者ID',
	`creator_name` varchar(64) NOT NULL DEFAULT '' COMMENT '创建者名称',
	`union_id` varchar(64) NOT NULL DEFAULT '' COMMENT '所有者关联组ID',
	`template_name` varchar(64) NOT NULL DEFAULT '' COMMENT '模板名',
	`filename` varchar(256) NOT NULL DEFAULT '' COMMENT '文件名',
	`title` varchar(64) NOT NULL DEFAULT '' COMMENT '任务标题',
	`md5` varchar(64) NOT NULL DEFAULT '' COMMENT '指纹',
	`status` enum('exporting','success','fail') NOT NULL DEFAULT 'exporting' COMMENT '任务状态',
	`timeout` varchar(15) NOT NULL DEFAULT '' COMMENT '任务处理超时时间',
	`size` int(11) unsigned NOT NULL DEFAULT '0' COMMENT '文件大小,单位B',
	`url` varchar(256) NOT NULL DEFAULT '' COMMENT '下载地址',
	`remark` varchar(256) NOT NULL DEFAULT '' COMMENT '备注',
	`expired_at` datetime DEFAULT NULL COMMENT '文件过期时间',
	`created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
	`updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
	PRIMARY KEY (`id`),
	UNIQUE KEY `uniq_md5` (`app_id`,`md5`),
	UNIQUE KEY `filename` (`filename`),
	KEY `idx_task` (`app_id`,`created_at`),
	KEY `idx_expired_at` (`expired_at`)

) ENGINE=InnoDB AUTO_INCREMENT=225 DEFAULT CHARSET=utf8mb4 COMMENT='下载任务表';
*/
var Export_export_task_table = sqlbuilder.NewTableConfig("t_export_task").AddColumns(
	sqlbuilder.NewColumn("id", sqlbuilder.GetField(NewId)),
	sqlbuilder.NewColumn("config_key", sqlbuilder.GetField(NewConfigKey)),
	sqlbuilder.NewColumn("app_id", sqlbuilder.GetField(NewAppId)),
	sqlbuilder.NewColumn("creator_id", sqlbuilder.GetField(NewCreatorId)),
	sqlbuilder.NewColumn("filename", sqlbuilder.GetField(NewFilename)),
	sqlbuilder.NewColumn("md5", sqlbuilder.GetField(NewMD5)),
	sqlbuilder.NewColumn("status", sqlbuilder.GetField(NewStatus)),
	sqlbuilder.NewColumn("timeout", sqlbuilder.GetField(NewTimeout)),
	sqlbuilder.NewColumn("url", sqlbuilder.GetField(NewUrl)),
	sqlbuilder.NewColumn("remark", sqlbuilder.GetField(NewRemark)),
	sqlbuilder.NewColumn("expired_at", sqlbuilder.GetField(NewExpiredAt)),
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

const (
	Task_status_success   = "success"
	Task_status_failed    = "fail"
	Task_status_exporting = "exporting"
)

type ExportTaskModel struct {
	Id        int    `gorm:"column:id"  json:"id"`
	ConfigKey string `gorm:"column:configKey"  json:"configKey"`
	AppId     string `gorm:"column:appId"  json:"appId"`
	CreatorId string `gorm:"column:creatorId"  json:"creatorId"`
	Filename  string `gorm:"column:filename"  json:"filename"`
	MD5       string `gorm:"column:md5"  json:"md5"`
	Status    string `gorm:"column:status"  json:"status"`
	Timeout   string `gorm:"column:timeout"  json:"timeout"`
	Url       string `gorm:"column:url"  json:"url"`
	Remark    string `gorm:"column:remark"  json:"remark"`
	ExpiredAt string `gorm:"column:expiredAt"  json:"expiredAt"`
	CreatedAt string `gorm:"column:createdAt"  json:"createdAt"`
	UpdatedAt string `gorm:"column:updatedAt"  json:"updatedAt"`
}

type ExportTaskModels []ExportTaskModel

func (ms ExportTaskModels) GetConfigKeys() (configKeys []string) {
	for _, m := range ms {
		configKeys = append(configKeys, m.ConfigKey)
	}
	return configKeys
}

func (ms ExportTaskModels) IsAllSuccessed() bool {
	for _, m := range ms {
		if m.Status != Task_status_success {
			return false
		}
	}
	return true
}

type ExportTaskRepository struct {
	table sqlbuilder.TableConfig
}

func NewExportTaskRepository(table sqlbuilder.TableConfig) *ExportTaskRepository {
	return &ExportTaskRepository{
		table: table,
	}
}

type ExportTaskRepositoryAddIn struct {
	AppId     string `json:"app_id"`
	CreatorId string `json:"creatorId"`
	Filename  string `json:"filename"`
	MD5       string `json:"md5"`
	Status    string `json:"status"`
	Timeout   string `json:"timeout"`
	Url       string `json:"url"`
	Remark    string `json:"remark"`
	ExpiredAt string `json:"expired_at"`
}

func (in ExportTaskRepositoryAddIn) Fields() sqlbuilder.Fields {
	return sqlbuilder.Fields{
		NewAppId(in.AppId).SetRequired(true),
		NewCreatorId(in.CreatorId).SetRequired(true),
		NewFilename(in.Filename).SetRequired(true),
		NewMD5(in.MD5).SetRequired(true),
		NewStatus(in.Status).SetRequired(true),
		NewTimeout(in.Timeout),
		NewUrl(in.Url).SetRequired(true),
		NewRemark(in.Remark),
		NewExpiredAt(in.ExpiredAt),
	}
}

func (s ExportTaskRepository) Add(in ExportTaskRepositoryAddIn) (id uint64, err error) {
	id, _, err = s.table.Repository().InsertWithLastId(in.Fields())
	if err != nil {
		return 0, err
	}
	return id, nil
}

type ExportTaskRepositoryUpdateStatusIn struct {
	Id     int    `json:"id"`
	Status string `json:"status"`
	Remark string `json:"remark"`
}

func (in ExportTaskRepositoryUpdateStatusIn) Fields() sqlbuilder.Fields {
	return sqlbuilder.Fields{
		NewId(in.Id).SetRequired(true).AppendWhereFn(sqlbuilder.ValueFnForward),
		NewStatus(in.Status).SetRequired(true),
		NewRemark(in.Remark),
	}
}

type ChangeStatus struct {
	EventId  string `json:"eventId"`
	Identity string `json:"identity"`
	Status   string `json:"status"`
}

const (
	ChangeStatus_EventId = "changeStatus"
)

func (e ChangeStatus) ToMessage() (msg *sqlbuilder.Message, err error) {
	return sqlbuilder.MakeMessage(e)
}

func (s ExportTaskRepository) UpdateStatus(in ExportTaskRepositoryUpdateStatusIn) (err error) {
	err = s.table.Repository().Update(in.Fields())
	if err != nil {
		return err
	}
	event := sqlbuilder.IdentityEvent{
		Operation:         ChangeStatus_EventId,
		IdentityValue:     cast.ToString(in.Id),
		IdentityFieldName: sqlbuilder.GetFieldName(NewId),
	}
	err = s.table.Publish(event)
	if err != nil {
		return err
	}
	return nil
}

func (s ExportTaskRepository) GetByIds(ids ...string) (models ExportTaskModels, err error) {
	fs := sqlbuilder.Fields{
		NewId(0).SetRequired(true).AppendWhereFn(sqlbuilder.ValueFnForward).Apply(func(f *sqlbuilder.Field, fs ...*sqlbuilder.Field) {
			f.ValueFns.ResetSetValueFn(func(inputValue any, f *sqlbuilder.Field, fs ...*sqlbuilder.Field) (any, error) {
				return ids, nil
			})
		}),
	}
	err = s.table.Repository().All(&models, fs)
	if err != nil {
		return nil, err
	}
	return models, nil
}

func (s ExportTaskRepository) GetByFilename(filenames ...string) (models ExportTaskModels, err error) {
	fs := sqlbuilder.Fields{
		NewFilename("").SetRequired(true).AppendWhereFn(sqlbuilder.ValueFnForward).Apply(func(f *sqlbuilder.Field, fs ...*sqlbuilder.Field) {
			f.ValueFns.ResetSetValueFn(func(inputValue any, f *sqlbuilder.Field, fs ...*sqlbuilder.Field) (any, error) {
				return filenames, nil
			})
		}),
	}
	err = s.table.Repository().All(&models, fs)
	if err != nil {
		return nil, err
	}
	return models, nil
}
