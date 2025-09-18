package excelrw

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type FetcherDataFn func(loopCount int, param map[string]any) (rows []map[string]string, forceBreak bool, err error) // 格式化请求参数、请求数据、返回数据 rows 为 []struct{} 或者 []map[string]any 格式
type CallBackFn func(params map[string]any) error                                                                   // 回调函数，用于处理数据导出后的后续操作

type ExportExcel struct {
	filename        string          // 文件名称可能和具体导出场景有关,如导出操作用户id，所以改成get/set 方式处理
	Titles          FieldMetas      `json:"titles"`
	Interval        time.Duration   `json:"interval"`
	DeleteFileDelay time.Duration   `json:"deleteFileDelay"`
	ErrorHandler    func(err error) // 处理错误
	FetcherDataFn   FetcherDataFn   // 格式化请求参数、请求数据、返回数据
	CallBackFn      CallBackFn      // 回调函数，用于处理数据导出后的后续操作
}

func NewExportExcel(filename string, titles FieldMetas) *ExportExcel {
	return &ExportExcel{filename: filename, Titles: titles}
}

func (exportExcel *ExportExcel) SetFilename(filename string) {
	exportExcel.filename = filename
}

func (exportExcel ExportExcel) GetFilename() (filename string) {
	return exportExcel.filename
}
func (exportExcel *ExportExcel) WithFetcherDataFn(fn FetcherDataFn) *ExportExcel {
	exportExcel.FetcherDataFn = fn
	return exportExcel
}

func (exportExcel ExportExcel) Export(params map[string]any) (excelFielname string, err error) {
	if exportExcel.filename == "" {
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
	ecw := NewExcelStreamWriter(ctx, exportExcel.filename, exportExcel.Titles)
	ecw = ecw.WithInterval(exportExcel.Interval).WithDeleteFile(exportExcel.DeleteFileDelay, exportExcel.ErrorHandler).WithFetcher(func(loopCount int) (rows []map[string]string, forceBreak bool, err error) {
		rows, forceBreak, err = exportExcel.FetcherDataFn(loopCount, params)
		if err != nil {
			return nil, forceBreak, err
		}
		return rows, forceBreak, nil
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
		err = exportExcel.CallBackFn(params)
		if err != nil {
			return "", err
		}
	}
	excelFielname = ecw.GetFilename()
	return excelFielname, nil
}
