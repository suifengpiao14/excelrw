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

func TestSliceAny2string(t *testing.T) {
	t.Run("mapSlice2string", func(t *testing.T) {

		ma := []map[string]any{
			{"name": "张三", "age": 18},
			{"name": "李四", "age": 20},
		}
		rows := excelrw.SliceAny2stringMust(ma)
		fmt.Println(rows)
	})

	t.Run("structSlice2string", func(t *testing.T) {
		us := []*User{
			{"张三", 18},
			{"李四", 20},
		}
		rows := excelrw.SliceAny2stringMust(us)
		fmt.Println(rows)
	})
	t.Run("any", func(t *testing.T) {
		ma := []map[string]any{
			{"name": "张三", "age": 18},
			{"name": "李四", "age": 20},
		}
		an := any(ma)
		rows := excelrw.SliceAny2stringMust(an)
		fmt.Println(rows)
	})
}
