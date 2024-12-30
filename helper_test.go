package excelrw_test

import (
	"fmt"
	"testing"

	"github.com/suifengpiao14/excelrw"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestMapAny2string(t *testing.T) {
	ma := []map[string]any{
		{"name": "张三", "age": 18},
		{"name": "李四", "age": 20},
	}
	rows := excelrw.MapAny2string(ma)
	fmt.Println(rows)
}

func TestStructSlice2mapRecords(t *testing.T) {
	users := []User{
		{Name: "张三", Age: 18},
		{Name: "李四", Age: 20},
	}
	rows := excelrw.StructSlice2mapRecords(users)
	fmt.Println(rows)
}
