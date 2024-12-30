package excelrw

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// MapAny2string 把map[string]any 转成 map[string]string
func MapAny2string(originalData []map[string]any) (newData []map[string]string) {
	for _, originRow := range originalData {
		row := make(map[string]string)
		for k, v := range originRow {
			row[k] = cast.ToString(v)

		}
		newData = append(newData, row)
	}
	return newData
}

// StructSlice2mapRecords 把结构体切片转成 map[string]string 格式的记录集
func StructSlice2mapRecords(structSlice any) (newData []map[string]string) {
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
		if i == 0 && v.Kind() != reflect.Struct { // 判断第一个即可
			err := errors.Errorf("required struct , but got :%T", originRow)
			panic(err)
		}
		for j := 0; j < v.NumField(); j++ {
			field := v.Type().Field(j)
			key := field.Name
			val := cast.ToString(v.Field(j).Interface())
			row[key] = val
		}
		newData = append(newData, row)
	}

	return newData
}
