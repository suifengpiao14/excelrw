package excelrw_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/suifengpiao14/excelrw"
	"github.com/xuri/excelize/v2"
)

func TestRead(t *testing.T) {
	filename := "./example/example.xlsx"
	fd, err := excelize.OpenFile(filename)
	require.NoError(t, err)
	reader := excelrw.NewExcelReader()
	sheet := "sheet1"
	fieldMap := map[string]string{
		"a": "Fsort",
		"b": "Ftype",
		"c": "Funique_code",
		"d": "Fposition_code",
		"e": "Fposition_name",
		"f": "Fclass_key",
		"g": "Fclass_name",
	}
	data, err := reader.Read(fd, sheet, fieldMap, 2, true)
	require.NoError(t, err)
	b, err := json.Marshal(data)
	require.NoError(t, err)
	fmt.Println(string(b))
}
