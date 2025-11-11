package repository

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cast"
	"github.com/suifengpiao14/apihttpprotocol"
	"github.com/suifengpiao14/httpraw"
	"github.com/suifengpiao14/sqlbuilder"
	"github.com/tidwall/gjson"
)

const (
	ReqeustLog_status_init = "init"
	//ReqeustLog_status_finished = "finished"
	ReqeustLog_result_success = "success"
	ReqeustLog_result_fail    = "fail"
)

var Request_log_table = sqlbuilder.NewTableConfig("t_request_log").AddColumns(
	sqlbuilder.NewColumn("Fid", sqlbuilder.GetField(NewId)),
	sqlbuilder.NewColumn("Fconfig_key", sqlbuilder.GetField(NewConfigKey)),
	sqlbuilder.NewColumn("Fdepend_task_id", sqlbuilder.GetField(NewDependTskId)),
	sqlbuilder.NewColumn("Frequest_dto", sqlbuilder.GetField(NewRequestDTO)),
	sqlbuilder.NewColumn("Fcurl", sqlbuilder.GetField(NewCURL)),
	sqlbuilder.NewColumn("Fresponse_dto", sqlbuilder.GetField(NewResponseDTO)),
	sqlbuilder.NewColumn("Fbusiness_code_path", sqlbuilder.GetField(NewBusinessCodePath)),
	sqlbuilder.NewColumn("Fbusiness_ok_code", sqlbuilder.GetField(NewBusinessOkCode)),
	sqlbuilder.NewColumn("Fhttp_code", sqlbuilder.GetField(NewHttpCode)),
	sqlbuilder.NewColumn("Ferror", sqlbuilder.GetField(NewError)),
	sqlbuilder.NewColumn("Fstatus", sqlbuilder.GetField(NewStatus)),
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

func SubscribeTaskFinishedEvent() (consumerMaker func(table sqlbuilder.TableConfig) (consumer sqlbuilder.Consumer)) {
	return func(table sqlbuilder.TableConfig) (consumer sqlbuilder.Consumer) {
		publishTable, err := table.GetTopicTable(Export_export_task_table)
		if err != nil {
			panic(err)
		}
		return sqlbuilder.MakeIdentityEventSubscriber(publishTable, func(model ExportTaskModel) (err error) {
			//如果状态为成功，则发起callback请求
			switch model.Status {
			case Task_status_success:
				requestService := NewRequestLogRepository(table)
				taskId := cast.ToString(model.Id)
				requests, err := requestService.GetByDependTaskId(taskId)
				if err != nil {
					return err
				}
				taskService := NewExportTaskRepository(publishTable)
				for _, request := range requests {
					reqDTO, err := request.ParseReqeuestDTO()
					if err != nil {
						updateIn := RequestLogRepositoryUpdateResponseIn{
							Id:     request.Id,
							Result: ReqeustLog_result_fail,
							Error:  err.Error(),
						}
						err = requestService.UpdateResponse(updateIn)
						if err != nil {
							return err
						}
						continue
					}

					dependTaskIds := strings.Split(request.DependTaskId, ",")
					models, err := taskService.GetByIds(dependTaskIds...)
					if err != nil {
						return err
					}
					isSuccessed := models.IsAllSuccessed()
					if isSuccessed {
						proxyRequest := ProxyRequest{
							ReqDTO:           *reqDTO,
							BusinessCodePath: request.BusinessCodePath,
							BusinessOkCode:   request.BusinessOkCode,
						}
						resp, err := proxyRequest.Request()
						if err != nil {
							resp.Result = ReqeustLog_result_fail
							resp.Error = err.Error()
						}
						responseIn := RequestLogRepositoryUpdateResponseIn{
							Id:          request.Id,
							ResponseDTO: resp.RespDTO.String(),
							HttpCode:    cast.ToString(resp.HttpCode),
							Result:      resp.Result,
							Error:       resp.Error,
						}
						err = requestService.UpdateResponse(responseIn)
						if err != nil {
							return err
						}
					}
				}

			case Task_status_failed:
				//todo 有任务失败，应该将依赖的回调函数也标记为失败
			}
			return nil
		})
	}
}

type ProxyRequest struct {
	ReqDTO           httpraw.RequestDTO `json:"reqDTO"`
	BusinessCodePath string             `json:"businessCodePath"` //业务成功标识路径，例如：$.code
	BusinessOkCode   string             `json:"businessOkCode"`   //业务成功标识值，例如：0
}

type ProxyResponse struct {
	HttpCode int                 `json:"httpCode"`
	RespDTO  httpraw.ResponseDTO `json:"respDTO"`
	Result   string              `json:"result"`
	Error    string              `json:"error"`
}

func (p ProxyRequest) Request() (proxyResponse ProxyResponse, err error) {
	requestDTO := p.ReqDTO
	client := apihttpprotocol.NewClientProtocol(requestDTO.Method, requestDTO.URL)
	client.Request().Headers = requestDTO.Headers.HttpHeaders() //设置头

	var resp json.RawMessage
	err = client.Do(requestDTO.Body, &resp)
	response := client.Response()
	responseDTO := httpraw.ResponseDTO{
		Headers: httpraw.HttpHeader2Headers(response.Headers),
		Body:    string(resp),
	}
	if err != nil {
		return proxyResponse, err
	}
	result := ReqeustLog_result_fail
	if p.BusinessCodePath != "" && p.BusinessOkCode != "" {
		//校验业务是否成功
		businessCode := gjson.GetBytes(resp, p.BusinessCodePath).String()

		if businessCode == p.BusinessOkCode {
			result = ReqeustLog_result_success
		} else {
			result = ReqeustLog_result_fail
		}
	}
	proxyResponse = ProxyResponse{
		RespDTO:  responseDTO,
		Result:   result,
		HttpCode: response.HttpCode,
	}
	return proxyResponse, nil

}

type RequestLog struct {
	Id               int    `gorm:"column:id"  json:"id"`
	ConfigId         string `gorm:"column:configId"  json:"configId"`
	DependTaskId     string `gorm:"column:dependTaskId"  json:"dependTaskId"`
	BusinessCodePath string `gorm:"column:businessCodePath"  json:"businessCodePath"`
	BusinessOkCode   string `gorm:"column:businessOkCode"  json:"businessOkCode"`
	RequestDTO       string `gorm:"column:requestDTO"  json:"requestDTO"`
	ResponseDTO      string `gorm:"column:responseDTO"  json:"responseDTO"`
	HttpCode         string `gorm:"column:httpCode"  json:"httpCode"`
	Result           string `gorm:"column:result"  json:"result"`
	CreatedAt        string `gorm:"column:createdAt" json:"createdAt"`
	UpdatedAt        string `gorm:"column:updatedAt" json:"updatedAt"`
}

func (r RequestLog) ParseReqeuestDTO() (reqDTO *httpraw.RequestDTO, err error) {
	reqDTO, err = httpraw.ParseRequestDTO(r.RequestDTO)
	if err != nil {
		return reqDTO, err
	}
	return reqDTO, nil
}

type RequestLogs []RequestLog

type RequestLogRepository struct {
	table sqlbuilder.TableConfig
}

func NewRequestLogRepository(table sqlbuilder.TableConfig) (repository *RequestLogRepository) {
	err := table.CheckMissOutFieldName(Request_log_table)
	if err != nil {
		panic(err)
	}
	table = table.WithConsumerMakers(SubscribeTaskFinishedEvent())
	err = table.Init() //启动消费者
	if err != nil {
		panic(err)
	}
	return &RequestLogRepository{
		table: table,
	}
}

type RequestLogRepositoryInsertIn struct {
	DependTskId      string `gorm:"column:dependTaskId"  json:"dependTaskId"`
	ConfigKey        string `gorm:"column:configKey"  json:"configKey"`
	RequestDTO       string `gorm:"column:requestDTO"  json:"requestDTO"`
	BusinessCodePath string `gorm:"column:businessCodePath"  json:"businessCodePath"`
	BusinessOkCode   string `gorm:"column:businessOkCode"  json:"businessOkCode"`
	CURL             string `gorm:"column:curl"  json:"curl"`
}

func (in RequestLogRepositoryInsertIn) Fields() sqlbuilder.Fields {
	return sqlbuilder.Fields{
		NewDependTskId(in.DependTskId).SetRequired(true),
		NewConfigKey(in.ConfigKey).SetRequired(true),
		NewRequestDTO(in.RequestDTO).SetRequired(true),
		NewBusinessCodePath(in.BusinessCodePath),
		NewBusinessOkCode(in.BusinessOkCode),
		NewCURL(in.CURL),
		NewStatus(ReqeustLog_status_init),
	}
}

func (r *RequestLogRepository) Insert(in RequestLogRepositoryInsertIn) (logId uint64, err error) {
	logId, _, err = r.table.Repository().InsertWithLastId(in.Fields())
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
	Error       string `gorm:"column:error"  json:"error"`
}

func (in RequestLogRepositoryUpdateResponseIn) Fields() sqlbuilder.Fields {
	return sqlbuilder.Fields{
		NewId(in.Id).SetRequired(true).AppendWhereFn(sqlbuilder.ValueFnForward),
		NewResponseDTO(in.ResponseDTO).SetRequired(true),
		NewHttpCode(in.HttpCode).SetRequired(true),
		NewResult(in.Result).SetRequired(true),
	}
}

func (r *RequestLogRepository) UpdateResponse(in RequestLogRepositoryUpdateResponseIn) (err error) {
	err = r.table.Repository().Update(in.Fields())
	if err != nil {
		return err
	}
	return nil
}

func (r *RequestLogRepository) GetByDependTaskId(dependTskId string) (logs RequestLogs, err error) {
	fs := sqlbuilder.Fields{
		NewDependTskId(dependTskId).SetRequired(true).AppendWhereFn(sqlbuilder.ValueFnFindInSet).SetDelayApply(func(f *sqlbuilder.Field, fs ...*sqlbuilder.Field) {
			columns := f.GetTable().Columns.DbNameWithAlias().AsAny()
			f.SetSelectColumns(columns...)
		}),
		NewStatus(ReqeustLog_status_init).AppendWhereFn(sqlbuilder.ValueFnForward),
	}
	err = r.table.Repository().All(&logs, fs)
	if err != nil {
		return nil, err
	}
	return logs, nil
}
