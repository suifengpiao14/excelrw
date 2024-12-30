package excelrw

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// SliceAny2string 将 []struct{}, []map[string]any 转成 []map[string]string
func SliceAny2string(structSlice any) (newData []map[string]string) {
	rv := reflect.Indirect(reflect.ValueOf(structSlice))
	if rv.Kind() != reflect.Slice {
		err := errors.Errorf("required struct slice, but got :%T", structSlice)
		panic(err)
	}

	if rv.Len() == 0 {
		return newData
	}

	for i := 0; i < rv.Len(); i++ {
		originRow := rv.Index(i).Interface()
		row := make(map[string]string)
		v := reflect.Indirect(reflect.ValueOf(originRow))
		switch v.Kind() {
		case reflect.Struct:
			for j := 0; j < v.NumField(); j++ {
				field := v.Type().Field(j)
				key := field.Tag.Get("json")
				if key == "" {
					key = field.Name
				}
				val := cast.ToString(v.Field(j).Interface())
				row[key] = val
			}
		case reflect.Map:
			for k, v := range originRow.(map[string]any) {
				row[k] = cast.ToString(v)
			}
		default:
			err := errors.Errorf("required struct or map , but got :%T", originRow)
			panic(err)

		}
		newData = append(newData, row)
	}

	return newData
}
