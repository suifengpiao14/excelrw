package excelrwhook // package 名称固定
import "github.com/suifengpiao14/apihttpprotocol"

/*
type HandlerFuncRequestMessage func(message *RequestMessage) (err error)
type HandlerFuncResponseMessage func(message *ResponseMessage) (err error)
*/

func RequestMiddleware(message *apihttpprotocol.RequestMessage) (err error) {
	//todo 此处编写请求中间件逻辑
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
	//todo 此处编写响应中间件逻辑
	return nil
}
