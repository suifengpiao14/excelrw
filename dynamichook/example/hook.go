package excelrwhook // package 名称固定
import (
	"fmt"

	"github.com/suifengpiao14/apihttpprotocol"
)

/*
type HandlerFuncRequestMessage func(message *RequestMessage) (err error)
type HandlerFuncResponseMessage func(message *ResponseMessage) (err error)
*/

func RequestMiddleware(message *apihttpprotocol.RequestMessage) (err error) {
	//todo 此处编写请求中间件逻辑
	fmt.Println(message.String())
	err = message.Next()
	if err != nil {
		return err
	}
	return nil

}

func ResponseMiddleware(message *apihttpprotocol.ResponseMessage) (err error) {
	err = message.Next()
	if err != nil {
		return err
	}
	fmt.Println(message.String())
	//todo 此处编写响应中间件逻辑
	return nil
}

/*
map[string]string ["_rowNumber": "1", "engineerEvaluatePrice": "150", "paymentTime": "", "createTime": "2025-10-29 16:54:05", "channelName": "闲鱼上门竞价", "productName": "iPhone 3GS", "cityId": "440300", "cityName": "深圳市", "status": "1", "statusText": "待使用", "adjustPrice": "0", "id": "65", "channelId": "10001485", "orderId": "16027", "userId": "1", "paymentAmount": "0", ]
*/

func RecordFormat(record map[string]string) (newRecord map[string]string, err error) {
	return record, nil
}
