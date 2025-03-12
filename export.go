package excelrw

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type FetcherDataFn func(currentPageIndex int, param map[string]any) (rows any, err error) // 格式化请求参数、请求数据、返回数据 rows 为 []struct{} 或者 []map[string]any 格式
type CallBackFn func(param map[string]any) (err error)                                    // 回调函数，用于处理数据导出后的后续操作

type ExportExcel struct {
	Filename        string          `json:"filename"`
	Titles          FieldMetas      `json:"titles"`
	Interval        time.Duration   `json:"interval"`
	DeleteFileDelay time.Duration   `json:"deleteFileDelay"`
	Params          map[string]any  `json:"params"`
	ErrorHandler    func(err error) // 处理错误
	FetcherDataFn   FetcherDataFn   // 格式化请求参数、请求数据、返回数据
	CallBackFn      CallBackFn      // 回调函数，用于处理数据导出后的后续操作

}

func (exportExcel ExportExcel) Export() (excelFielname string, err error) {
	if exportExcel.Filename == "" {
		err = errors.Errorf("filename required")
		return "", err
	}
	if len(exportExcel.Titles) == 0 {
		err = errors.Errorf("titles required")
		return "", err
	}
	if exportExcel.FetcherDataFn == nil {
		err = errors.Errorf("FetcherDataFn required")
		return "", err
	}

	ctx := context.Background()
	ecw := NewExcelStreamWriter(ctx, exportExcel.Filename, exportExcel.Titles)
	ecw = ecw.WithInterval(exportExcel.Interval).WithDeleteFile(exportExcel.DeleteFileDelay, exportExcel.ErrorHandler).WithFetcher(func(prevPageIndex int) (curentPageIndex int, rows []map[string]string, err error) {
		curentPageIndex = prevPageIndex + 1
		if curentPageIndex < 0 {
			curentPageIndex = 0
		}
		records, err := exportExcel.FetcherDataFn(curentPageIndex, exportExcel.Params)
		if err != nil {
			return curentPageIndex, nil, err
		}
		rows, err = SliceAny2string(records)
		if err != nil {
			return curentPageIndex, nil, err
		}

		return curentPageIndex, rows, nil
	})
	errChan, err := ecw.Run()
	if err != nil {
		return "", err
	}
	err = <-errChan
	if err != nil {
		return "", err
	}
	if exportExcel.CallBackFn != nil {
		err = exportExcel.CallBackFn(exportExcel.Params)
		if err != nil {
			return "", err
		}
	}
	excelFielname = ecw.GetFilename()
	return excelFielname, err
}
