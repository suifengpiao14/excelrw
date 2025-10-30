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
	ReqeustMiddlewareName  string               `json:"beforeRequestFuncName"`
	ResponseMiddlewareName string               `json:"afterRequestFuncName"`
	DynamicExtension       *yaegijson.Extension `json:"-"`
}

type DynamicMiddleware struct {
	RequestMiddleware  apihttpprotocol.HandlerFuncRequestMessage
	ResponseMiddleware apihttpprotocol.HandlerFuncResponseMessage
}

func (p DynamicHook) MakeMiddleware() (out DynamicMiddleware, err error) {
	// 动态编译扩展代码
	extension := p.DynamicExtension
	if extension == nil {
		err = errors.Errorf("DynamicExtensionHttpRaw is nil")
		return out, err

	}
	err = extension.GetDestFuncImpl(p.ReqeustMiddlewareName, &out.RequestMiddleware)
	if err != nil && errors.Is(err, yaegijson.Error_not_found_func) {
		err = nil
	}
	if err != nil {
		return out, err
	}
	err = extension.GetDestFuncImpl(p.ResponseMiddlewareName, &out.ResponseMiddleware)
	if err != nil && errors.Is(err, yaegijson.Error_not_found_func) {
		err = nil
	}
	if err != nil {
		return out, err
	}

	return out, nil
}

func NewExtension() *yaegijson.Extension {
	extension := yaegijson.NewExtension().WithSymbols(Symbols)
	return extension
}
