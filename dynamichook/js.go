package dynamichook

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/suifengpiao14/excelrw/defined"
	"github.com/suifengpiao14/httpraw"
)

var ErrorJSNotFound = fmt.Errorf("RecordFormatFn function not found")

type JSVM struct {
	vm *goja.Runtime
}

func ParseJSVM(jsScript string) (jsvm *JSVM, err error) {
	vm := goja.New()
	_, err = vm.RunString(jsScript)
	if err != nil {
		return nil, err
	}
	JSVM := &JSVM{
		vm: vm,
	}
	return JSVM, nil
}

func (vm *JSVM) RunString(jsScript string) (err error) {
	_, err = vm.vm.RunString(jsScript)
	if err != nil {
		err = errors.WithMessagef(err, "RunString error: %s", jsScript)
		return err
	}
	return nil
}

func (jsVm *JSVM) RecordFormatFn(fnName string) (fn defined.RecordFormatFn, err error) {
	fn = func(record map[string]string) (newRecord map[string]string, err error) { // 确保一定有默认值，减少调用方nil判断的bug（比如调用方忽略ErrorJSNotFound 错误，直接使用fn）
		return record, nil
	}
	vm := jsVm.vm
	jsFuncVal := vm.Get(fnName)
	if jsFuncVal == nil {
		err = errors.WithMessagef(ErrorJSNotFound, "function:%s", fnName)
		return fn, err
	}

	jsFunc, ok := goja.AssertFunction(jsFuncVal)
	if !ok {
		return fn, fmt.Errorf("RecordFormatFn%s is not a function", fnName)
	}

	// 封装成 Go 函数
	fn = func(record map[string]string) (map[string]string, error) {
		jsRecord := vm.ToValue(record)
		res, err := jsFunc(goja.Undefined(), jsRecord)
		if err != nil {
			return nil, fmt.Errorf("RecordFormatFn js execution error: %w", err)
		}

		var newRecord map[string]string
		if err := vm.ExportTo(res, &newRecord); err != nil {
			return nil, fmt.Errorf("RecordFormatFn export js result error: %w", err)
		}
		return newRecord, nil
	}

	return fn, nil

}

func (jsVm *JSVM) RequestHook(fnName string, requestDTO httpraw.RequestDTO) (newRequestDTO *httpraw.RequestDTO, err error) {
	newRequestDTO = &requestDTO // 填充默认值
	vm := jsVm.vm
	jsFuncVal := vm.Get(fnName)
	if jsFuncVal == nil {
		err = errors.WithMessagef(ErrorJSNotFound, "function:%s", fnName)
		return nil, err
	}

	jsFunc, ok := goja.AssertFunction(jsFuncVal)
	if !ok {
		err = fmt.Errorf("RequestHook%s is not a function", fnName)
		return nil, err
	}

	jsRequestDTO := vm.ToValue(requestDTO)
	res, err := jsFunc(goja.Undefined(), jsRequestDTO)
	if err != nil {
		err = errors.WithMessage(err, "RequestHook js execution error")
		return nil, err
	}
	var newRequestDTO2 httpraw.RequestDTO
	if err := vm.ExportTo(res, &newRequestDTO2); err != nil {
		err = errors.WithMessage(err, "RequestHook export js result error")
		return newRequestDTO, err
	}
	return &newRequestDTO2, nil
}
