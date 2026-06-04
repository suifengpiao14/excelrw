package dynamichook

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/suifengpiao14/excelrw/defined"
	"github.com/suifengpiao14/httpraw"
)

var ErrorJSNotFound = fmt.Errorf("RecordFormatFn function not found")

type JSVM struct {
	vm *goja.Runtime
	mu sync.Mutex // 保护 vm 的并发访问
}

var (
	jsvmCacheMu sync.RWMutex
	jsvmCache   = make(map[string]*JSVM)
)

func scriptHash(jsScript string) string {
	h := md5.Sum([]byte(jsScript))
	return fmt.Sprintf("%x", h)
}

func ParseJSVM(jsScript string) (jsvm *JSVM, err error) {
	key := scriptHash(jsScript)

	// 快速路径：读锁查缓存
	jsvmCacheMu.RLock()
	cached, ok := jsvmCache[key]
	jsvmCacheMu.RUnlock()
	if ok {
		return cached, nil
	}

	// 慢路径：创建新 JSVM
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	registerUtils(vm)

	_, err = vm.RunString(jsScript)
	if err != nil {
		return nil, err
	}
	jsvm = &JSVM{
		vm: vm,
	}

	// 写入缓存（double-check 防止重复创建）
	jsvmCacheMu.Lock()
	if existing, ok := jsvmCache[key]; ok {
		jsvm = existing
	} else {
		jsvmCache[key] = jsvm
	}
	jsvmCacheMu.Unlock()

	return jsvm, nil
}

func (jsVm *JSVM) RunString(jsScript string) (err error) {
	jsVm.mu.Lock()
	defer jsVm.mu.Unlock()
	_, err = jsVm.vm.RunString(jsScript)
	if err != nil {
		err = errors.WithMessagef(err, "RunString error: %s", jsScript)
		return err
	}
	return nil
}

func (jsVm *JSVM) RecordFormatFn(fnName string) (fn defined.RecordFormatFn, err error) {
	fn = func(record map[string]string) (newRecord map[string]string, err error) {
		return record, nil
	}
	vm := jsVm.vm
	jsVm.mu.Lock()
	jsFuncVal := vm.Get(fnName)
	jsVm.mu.Unlock()
	if jsFuncVal == nil {
		err = errors.WithMessagef(ErrorJSNotFound, "function:%s", fnName)
		return fn, err
	}

	jsFunc, ok := goja.AssertFunction(jsFuncVal)
	if !ok {
		return fn, fmt.Errorf("RecordFormatFn%s is not a function", fnName)
	}

	fn = func(record map[string]string) (map[string]string, error) {
		jsVm.mu.Lock()
		defer jsVm.mu.Unlock()
		jsRecord := vm.ToValue(record)
		res, err := jsFunc(goja.Undefined(), jsRecord)
		if err != nil {
			return nil, fmt.Errorf("RecordFormatFn js execution error: %w", err)
		}

		var newRecordAny map[string]any
		if err := vm.ExportTo(res, &newRecordAny); err != nil {
			return nil, fmt.Errorf("RecordFormatFn export js result error: %w", err)
		}

		newRecord := make(map[string]string)
		for k, v := range newRecordAny {
			newRecord[k] = cast.ToString(v)
		}

		return newRecord, nil
	}

	return fn, nil

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

	jsFunc, err := jsVm.GetJSFn(fnName)
	if err != nil {
		err = errors.WithMessage(err, "ResponseFormatFn GetJSFn error")
		return fn, err
	}

	fn = func(responseDTO httpraw.ResponseDTO) (records []map[string]any, err error) {
		records = make([]map[string]any, 0)
		err = jsVm.CallJsFn(jsFunc, responseDTO, &records)
		if err != nil {
			err = errors.WithMessage(err, "ResponseFormatFn CallJsFn error")
			return records, err
		}
		return records, nil
	}
	return fn, nil

}

func (jsVm *JSVM) RequestFormatFn(fnName string) (fn defined.RequestFormatFn, err error) {
	fn = func(requestDTO httpraw.RequestDTO) (httpraw.RequestDTO, error) {
		return requestDTO, nil
	}
	jsFunc, err := jsVm.GetJSFn(fnName)
	if err != nil {
		err = errors.WithMessage(err, "RequestFormatFn GetJSFn error")
		return fn, err
	}
	fn = func(requestDTO httpraw.RequestDTO) (httpraw.RequestDTO, error) {
		var newRequestDTO2 httpraw.RequestDTO
		err = jsVm.CallJsFn(jsFunc, requestDTO, &newRequestDTO2)
		if err != nil {
			err = errors.WithMessage(err, "RequestFormatFn CallJsFn error")
			return requestDTO, err
		}
		return newRequestDTO2, nil
	}
	return fn, nil

}

func (jsVm *JSVM) SettingFn(fnName string) (fn defined.SettingFn, err error) {
	fn = func(body string) (Setting defined.Setting, err error) {
		setting := defined.Setting{}
		return setting, nil
	}
	jsFunc, err := jsVm.GetJSFn(fnName)
	if err != nil {
		err = errors.WithMessage(err, "SettingFn GetJSFn error")
		return fn, err
	}
	fn = func(body string) (setting defined.Setting, err error) {
		setting = defined.Setting{
			Filename: "",
			Titles:   defined.FieldMetas{},
		}
		err = jsVm.CallJsFn(jsFunc, body, &setting)
		if err != nil {
			err = errors.WithMessage(err, "SettingFn CallJsFn error")
			return setting, err
		}
		if setting.Titles == nil {
			setting.Titles = defined.FieldMetas{}
		}
		return setting, nil
	}
	return fn, nil
}
func (jsVm *JSVM) GetJSFn(fnName string) (jsFunc goja.Callable, err error) {
	vm := jsVm.vm
	jsVm.mu.Lock()
	jsFuncVal := vm.Get(fnName)
	jsVm.mu.Unlock()
	if jsFuncVal == nil {
		err = errors.WithMessagef(ErrorJSNotFound, "GetJSFn function:%s", fnName)
		return nil, err
	}
	jsFunc, ok := goja.AssertFunction(jsFuncVal)
	if !ok {
		err = fmt.Errorf("GetJSFn %s is not a function", fnName)
		return nil, err
	}
	return jsFunc, nil
}

func (jsVm *JSVM) CallJsFn(jsFunc goja.Callable, input any, output any) (err error) {
	vm := jsVm.vm
	var inputAny any = input
	if input == nil {
		inputAny = map[string]any{}
	} else {
		b, jsonErr := json.Marshal(input)
		if jsonErr != nil {
			return errors.WithMessage(jsonErr, "CallJsFn json marshal input error")
		}
		var tmp any
		jsonErr = json.Unmarshal(b, &tmp)
		if jsonErr != nil {
			return errors.WithMessage(jsonErr, "CallJsFn json unmarshal input error")
		}
		inputAny = tmp
	}

	jsVm.mu.Lock()
	defer jsVm.mu.Unlock()

	jsBody := vm.ToValue(inputAny)
	res, err := jsFunc(goja.Undefined(), jsBody)
	if err != nil {
		err = errors.WithMessage(err, "CallJs js execution error")
		return err
	}
	var result any
	if err := vm.ExportTo(res, &result); err != nil {
		err = errors.WithMessage(err, "CallJs export js result error")
		return err
	}

	if result != nil {
		b, err := json.Marshal(result)
		if err != nil {
			err = errors.WithMessage(err, "CallJsFn result json marshal error")
			return err
		}
		err = json.Unmarshal(b, output)
		if err != nil {
			err = errors.WithMessage(err, "CallJsFn result json unmarshal error")
			return err
		}
	}
	return nil
}

// registerUtils 把 md5 / base64 函数注册到 goja VM
func registerUtils(vm *goja.Runtime) {
	// md5(str) -> hex string
	vm.Set("md5", func(fc goja.FunctionCall) goja.Value {
		if len(fc.Arguments) < 1 {
			panic(vm.NewTypeError("md5 requires 1 argument"))
		}
		s := fc.Argument(0).String()
		sum := md5.Sum([]byte(s))
		hex := fmt.Sprintf("%x", sum)
		return vm.ToValue(hex)
	})

	// base64Encode(str) -> base64 string
	vm.Set("base64Encode", func(fc goja.FunctionCall) goja.Value {
		if len(fc.Arguments) < 1 {
			panic(vm.NewTypeError("base64Encode requires 1 argument"))
		}
		s := fc.Argument(0).String()
		enc := base64.StdEncoding.EncodeToString([]byte(s))
		return vm.ToValue(enc)
	})

	// base64Decode(b64str) -> decoded string (throws if invalid base64)
	vm.Set("base64Decode", func(fc goja.FunctionCall) goja.Value {
		if len(fc.Arguments) < 1 {
			panic(vm.NewTypeError("base64Decode requires 1 argument"))
		}
		s := fc.Argument(0).String()
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(string(b))
	})

	console := map[string]func(goja.FunctionCall) goja.Value{
		"log": func(fc goja.FunctionCall) goja.Value {
			var out strings.Builder
			for i, a := range fc.Arguments {
				if i > 0 {
					out.WriteString(" ")
				}
				out.WriteString(a.String())
			}
			if LogFn == nil {
				LogFn = defaultLogFn
			}
			s := fmt.Sprintf(`javascript:console.log:%s`, out.String())
			LogFn(s)
			return goja.Undefined()
		},
	}
	vm.Set("console", console)
}

var LogFn func(v ...any) = defaultLogFn
var defaultLogFn = func(v ...any) {
	fmt.Println(v...)
}
