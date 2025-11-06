package dynamichook

import (
	"encoding/json"
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

func (jsVm *JSVM) ResponseFormatFn(fnName string) (fn defined.ResponseFormatFn, err error) {
	fn = func(responseDTO httpraw.ResponseDTO) (records []map[string]any, err error) {
		records = make([]map[string]any, 0)
		if json.Valid([]byte(responseDTO.Body)) {
			err = json.Unmarshal([]byte(responseDTO.Body), &records)
			if err != nil {
				err = errors.WithMessagef(err, "json string:%s", responseDTO.Body)
				return nil, err
			}
			return records, nil
		}
		return nil, nil
	}
	vm := jsVm.vm
	jsFuncVal := vm.Get(fnName)
	if jsFuncVal == nil {
		err = errors.WithMessagef(ErrorJSNotFound, "function:%s", fnName)
		return fn, err
	}

	jsFunc, ok := goja.AssertFunction(jsFuncVal)
	if !ok {
		return fn, fmt.Errorf("ResponseFormatFn%s is not a function", fnName)
	}

	// 封装成 Go 函数
	fn = func(responseDTO httpraw.ResponseDTO) (records []map[string]any, err error) {
		jsResponseDTO := vm.ToValue(responseDTO)
		res, err := jsFunc(goja.Undefined(), jsResponseDTO)
		if err != nil {
			return nil, fmt.Errorf("ResponseFormatFn js execution error: %w", err)
		}

		records = make([]map[string]any, 0)
		if err := vm.ExportTo(res, &records); err != nil {
			return nil, fmt.Errorf("ResponseFormatFn export js result error: %w", err)
		}
		return records, nil
	}

	return fn, nil

}

func (jsVm *JSVM) RequestFormatFn(fnName string) (fn defined.RequestFormatFn, err error) {
	fn = func(requestDTO httpraw.RequestDTO) (httpraw.RequestDTO, error) {
		return requestDTO, nil
	}
	vm := jsVm.vm
	jsFuncVal := vm.Get(fnName)
	if jsFuncVal == nil {
		err = errors.WithMessagef(ErrorJSNotFound, "function:%s", fnName)
		return fn, err
	}

	jsFunc, ok := goja.AssertFunction(jsFuncVal)
	if !ok {
		err = fmt.Errorf("RequestFormatFn%s is not a function", fnName)
		return fn, err
	}
	fn = func(requestDTO httpraw.RequestDTO) (httpraw.RequestDTO, error) {
		jsRequestDTO := vm.ToValue(requestDTO)
		res, err := jsFunc(goja.Undefined(), jsRequestDTO)
		if err != nil {
			err = errors.WithMessage(err, "RequestFormatFn js execution error")
			return requestDTO, err
		}
		var newRequestDTO2 httpraw.RequestDTO
		if err := vm.ExportTo(res, &newRequestDTO2); err != nil {
			err = errors.WithMessage(err, "RequestFormatFn export js result error")
			return requestDTO, err
		}
		return newRequestDTO2, nil
	}
	return fn, nil

}
