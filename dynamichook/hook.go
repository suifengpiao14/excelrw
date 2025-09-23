package dynamichook

import (
	"github.com/pkg/errors"
	"github.com/suifengpiao14/apihttpprotocol"
	"github.com/suifengpiao14/yaegijson"
)

var Symbols = yaegijson.Symbols

//go:generate go install github.com/traefik/yaegi/cmd/yaegi
//go:generate yaegi extract github.com/suifengpiao14/apihttpprotocol

type DynamicHook struct {
	ReqeustMiddlewareName   string               `json:"beforeRequestFuncName"`
	ResponseMiddlewareName  string               `json:"afterRequestFuncName"`
	DynamicExtensionHttpRaw *yaegijson.Extension `json:"-"`
}

func (p DynamicHook) MakeMiddleware() (reqeustMiddleware apihttpprotocol.HandlerFuncRequestMessage, responseMiddleware apihttpprotocol.HandlerFuncResponseMessage, err error) {
	// 动态编译扩展代码
	extension := p.DynamicExtensionHttpRaw
	if extension == nil {
		err = errors.Errorf("DynamicExtensionHttpRaw is nil")
		return nil, nil, err

	}
	err = extension.GetDestFuncImpl(p.ReqeustMiddlewareName, &reqeustMiddleware)
	if err != nil {
		return nil, nil, err
	}
	err = extension.GetDestFuncImpl(p.ResponseMiddlewareName, &responseMiddleware)
	if err != nil {
		return nil, nil, err
	}
	return reqeustMiddleware, responseMiddleware, nil
}

func NewExtension() *yaegijson.Extension {
	extension := yaegijson.NewExtension().WithSymbols(Symbols)
	return extension
}
