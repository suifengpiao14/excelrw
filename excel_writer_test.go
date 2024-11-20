package excelrw_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/suifengpiao14/excelrw"
)

func TestWriteWithChan(t *testing.T) {
	data := make([]map[string]string, 0)

	err := json.Unmarshal([]byte(jsonData), &data)

	require.NoError(t, err)
	filename := "./example/example.xlsx"
	fieldMetas := excelrw.FieldMetas{
		{Name: "Fsort", Title: "排序"},
		{Name: "Ftype", Title: "类型"},
		{Name: "Funique_code", Title: "唯一值"},
		{Name: "Fposition_code", Title: "位置"},
		{Name: "Fposition_name", Title: "位置名称"},
		{Name: "Fclass_key", Title: "分类key"},
		{Name: "Fclass_name", Title: "分类名称"},
	}
	ctx := context.Background()
	ecw := excelrw.NewExcelStreamWriter(ctx, filename, fieldMetas)
	ecw.WithFetcher(func(loopIndex int) (currentPageIndex int, rows []map[string]string, err error) {
		return 0, data, nil
	})
	_, err = ecw.Run()
	require.NoError(t, err)
}

var jsonData = `
[
  {
    "Ftype": "12",
    "Funique_code": "camera_front",
    "Fposition_code": "camera_front",
    "Fposition_name": "正面",
    "Fclass_key": "key_camera",
    "Fclass_name": "相机",
    "Fsort": "1"
  },
  {
    "Ftype": "12",
    "Funique_code": "camera_total",
    "Fposition_code": "camera_total",
    "Fposition_name": "整机（包含配件）",
    "Fclass_key": "key_camera",
    "Fclass_name": "相机",
    "Fsort": "2"
  },
  {
    "Ftype": "12",
    "Funique_code": "camera_left",
    "Fposition_code": "camera_left",
    "Fposition_name": "左侧面",
    "Fclass_key": "key_camera",
    "Fclass_name": "相机",
    "Fsort": "3"
  },
  {
    "Ftype": "12",
    "Funique_code": "camera_right",
    "Fposition_code": "camera_right",
    "Fposition_name": "右侧面",
    "Fclass_key": "key_camera",
    "Fclass_name": "相机",
    "Fsort": "4"
  },
  {
    "Ftype": "12",
    "Funique_code": "camera_top",
    "Fposition_code": "camera_top",
    "Fposition_name": "顶部",
    "Fclass_key": "key_camera",
    "Fclass_name": "相机",
    "Fsort": "5"
  },
  {
    "Ftype": "12",
    "Funique_code": "camera_bottom",
    "Fposition_code": "camera_bottom",
    "Fposition_name": "底部",
    "Fclass_key": "key_camera",
    "Fclass_name": "相机",
    "Fsort": "6"
  }
]
`
