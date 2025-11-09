package repository

import "github.com/suifengpiao14/sqlbuilder"

var Request_log_table = sqlbuilder.NewTableConfig("t_request_log").AddColumns(
	sqlbuilder.NewColumn("Fid", sqlbuilder.GetField(NewId)),
	sqlbuilder.NewColumn("Fconfig_id", sqlbuilder.GetField(NewConfigId)),
	sqlbuilder.NewColumn("Frequest_dto", sqlbuilder.GetField(NewRequestDTO)),
	sqlbuilder.NewColumn("Fresponse_dto", sqlbuilder.GetField(NewResponseDTO)),
	sqlbuilder.NewColumn("Fhttp_code", sqlbuilder.GetField(NewHttpCode)),
	sqlbuilder.NewColumn("Fresult", sqlbuilder.GetField(NewResult)),
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

type RequestLog struct {
	Id          int    `gorm:"column:id"  json:"id"`
	ConfigId    int    `gorm:"column:configId"  json:"configId"`
	RequestDTO  string `gorm:"column:requestDTO"  json:"requestDTO"`
	ResponseDTO string `gorm:"column:responseDTO"  json:"responseDTO"`
	HttpCode    string `gorm:"column:httpCode"  json:"httpCode"`
	Result      string `gorm:"column:result"  json:"result"`
	CreatedAt   string `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt   string `gorm:"column:updatedAt" json:"updatedAt"`
}

type RequestLogs []RequestLog

type RequestLogRepository struct {
	table sqlbuilder.TableConfig
}

func NewRequestLogRepository() (repository *RequestLogRepository) {
	return &RequestLogRepository{
		table: Request_log_table,
	}
}

type RequestLogRepositoryInsertIn struct {
	ConfigId   int    `gorm:"column:configId"  json:"configId"`
	RequestDTO string `gorm:"column:requestDTO"  json:"requestDTO"`
}

func (in RequestLogRepositoryInsertIn) Fields() sqlbuilder.Fields {
	return sqlbuilder.Fields{
		NewConfigId(in.ConfigId),
		NewRequestDTO(in.RequestDTO),
	}
}

func (r *RequestLogRepository) Insert(in RequestLogRepositoryInsertIn) (logId int, err error) {
	err = r.table.Repository().Insert(in.Fields())
	if err != nil {
		return 0, err
	}
	return logId, nil
}

type RequestLogRepositoryUpdateResponseIn struct {
	Id          int    `gorm:"column:id"  json:"id"`
	ResponseDTO string `gorm:"column:responseDTO"  json:"responseDTO"`
	HttpCode    string `gorm:"column:httpCode"  json:"httpCode"`
	Result      string `gorm:"column:result"  json:"result"`
}

func (in RequestLogRepositoryUpdateResponseIn) Fields() sqlbuilder.Fields {
	return sqlbuilder.Fields{
		NewId(in.Id).SetRequired(true).AppendWhereFn(sqlbuilder.ValueFnForward),
		NewResponseDTO(in.ResponseDTO).SetRequired(true),
		NewHttpCode(in.HttpCode).SetRequired(true),
		NewResult(in.Result).SetRequired(true),
	}
}

func (r *RequestLogRepository) UpdateResponse(in RequestLogRepositoryInsertIn) (err error) {
	err = r.table.Repository().Update(in.Fields())
	if err != nil {
		return err
	}
	return nil
}
