package excelrw

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"
)

type FieldMeta struct {
	ColNumber int    `json:"colNumber"` // 列号(数字,1开始) 增加json tag,可方便调用方验证输入是否符合格式
	Name      string `json:"name"`      // 列名称
	Title     string `json:"title"`     // 列标题
}

type FieldMetas []FieldMeta

func (a FieldMetas) Len() int           { return len(a) }
func (a FieldMetas) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a FieldMetas) Less(i, j int) bool { return a[i].ColNumber < a[j].ColNumber }

func (fs *FieldMetas) Sort() {
	fs.InitColIndex()
	sort.Sort(fs)
}

// InitColIndex 默认使用序号作为colNumber
func (fs *FieldMetas) InitColIndex() {
	for i, fieldMeta := range *fs {
		if fieldMeta.ColNumber < 1 {
			(*fs)[i].ColNumber = i + 1
		}
	}
}

// 最小列
func (fs FieldMetas) MinColIndex() (minColIndex int) {
	sort.Sort(fs)
	if len(fs) > 0 {
		return fs[0].ColNumber
	}
	return 1 // 最小从1开始
}

// ValidateFieldMetas 验证字符串是否符合 FieldMetas 格式,供调用方接收入参时验证
func ValidateFieldMetas(fieldMetasStr string) (err error) {
	fieldMetas := make(FieldMetas, 0)
	err = json.Unmarshal([]byte(fieldMetasStr), &fieldMetas)
	if err != nil {
		err = errors.WithMessage(err, `excepted format:[{"colNumber":1,"name":"A","Title":"标题"}] ,colNumber/name at least one`)
		return err
	}
	return nil
}

type _ExcelWriter struct{}

func NewExcelWriter() (writer *_ExcelWriter) {
	return &_ExcelWriter{}
}

// GetRowNumber  获取下一次可以写入的行号
func (excelWriter *_ExcelWriter) GetRowNumber(fd *excelize.File, sheet string) (rowNumber int, err error) {
	rows, err := fd.Rows(sheet)
	if err != nil {
		return
	}
	cur := 0
	for rows.Next() {
		cur++
	}
	rowNumber = cur + 1
	return
}

// 获取excel文件，不存在则创建
func (excelWriter *_ExcelWriter) GetFile(filename string, sheet string, removeOldFile bool) (fd *excelize.File, err error) {
	if fileExists(filename) && removeOldFile {
		err = os.Remove(filename)
		if err != nil {
			return nil, err
		}
	}

	if !fileExists(filename) { // 不存在，创建文件
		err = os.MkdirAll(filepath.Dir(filename), os.ModePerm)
		if err != nil {
			return nil, err
		}
		tmpFd := excelize.NewFile()
		err = tmpFd.SaveAs(filename)
		if err != nil {
			return nil, err
		}
	}

	fd, err = excelize.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	index, err := fd.GetSheetIndex(sheet)
	if err != nil || index < 0 {
		index, err = fd.NewSheet(sheet)
		if err != nil {
			return nil, err
		}
	}
	fd.SetActiveSheet(index)
	return fd, nil
}

// 移除一行
func (excelWriter *_ExcelWriter) RemoveRow(fd *excelize.File, sheet string, row int) (err error) {
	err = fd.RemoveRow(sheet, row)
	return
}

// WriteAll  一次性全部写入文件（只负责写入，不负责保存和关闭文件，外部需要调用fd.Save()方法）
func (excelWriter *_ExcelWriter) WriteAll(fd *excelize.File, sheet string, fieldMetas FieldMetas, data ExchangeData) (nextRowNumber int, err error) {
	streamWriter, _, err := excelWriter.getStreamWriter(fd, sheet)
	if err != nil {
		return
	}
	defer func() {
		err = streamWriter.Flush()
		if err != nil {
			return
		}
	}()
	fieldMetas.Sort()
	minColIndex := fieldMetas.MinColIndex()
	colLen := fieldMetas.Len()
	rowNumber := data.RowNumber
	for _, record := range data.Data {
		// 组装一行数据
		row := make([]any, colLen)
		for _, fieldMeta := range fieldMetas {
			k := fieldMeta.ColNumber - 1
			row[k] = record[fieldMeta.Name]
		}

		// 获取当前行开始写入单元地址
		cell, err := excelize.CoordinatesToCellName(minColIndex, rowNumber)
		if err != nil {
			return 0, err
		}
		//写入一行数据
		err = streamWriter.SetRow(cell, row)
		if err != nil {
			return 0, err
		}
		rowNumber++ // 增加行号
	}
	nextRowNumber = rowNumber
	return
}

// Write2streamWriter 向写入流中写入数据
func (excelWriter *_ExcelWriter) Write2streamWriter(streamWriter *excelize.StreamWriter, fieldMetas FieldMetas, chanelData ExchangeData) (nextRowNumber int, err error) {
	fieldMetas.Sort()
	colLen := fieldMetas.Len()
	minColIndex := fieldMetas.MinColIndex() // 找到最小的列
	rowNumber := chanelData.RowNumber
	for _, record := range chanelData.Data {
		// 组装一行数据
		row := make([]any, colLen)
		for _, fieldMeta := range fieldMetas {
			k := fieldMeta.ColNumber - 1
			row[k] = record[fieldMeta.Name]
		}

		// 获取当前行开始写入单元地址
		cell, err := excelize.CoordinatesToCellName(minColIndex, rowNumber)
		if err != nil {
			return 0, err
		}
		//写入一行数据
		err = streamWriter.SetRow(cell, row)
		if err != nil {
			return 0, err
		}
		rowNumber++ // 增加行号
	}
	nextRowNumber = rowNumber
	return
}

// getStreamWriter 打开文件流，将已有的数据填写到流内，返回写入流
func (excelWriter *_ExcelWriter) getStreamWriter(fd *excelize.File, sheet string) (streamWriter *excelize.StreamWriter, nextRowNumber int, err error) {
	streamWriter, err = fd.NewStreamWriter(sheet)
	if err != nil {
		return
	}

	rows, err := fd.GetRows(sheet) //获取行内容
	if err != nil {
		return
	}
	//将源文件内容先写入excel
	rowNumber := 0
	for rowindex, oldRow := range rows {
		rowNumber = rowindex + 1
		colLen := len(oldRow)
		newRow := make([]any, colLen)
		for colIndex := 0; colIndex < colLen; colIndex++ {
			if oldRow == nil {
				newRow[colIndex] = nil
			} else {
				newRow[colIndex] = oldRow[colIndex]
			}
		}
		beginCell, _ := excelize.CoordinatesToCellName(1, rowNumber)
		err = streamWriter.SetRow(beginCell, newRow)
		if err != nil {
			return
		}
	}
	nextRowNumber = rowNumber + 1
	return streamWriter, nextRowNumber, nil
}

// ExchangeData 程序和excel文件数据交换媒介
type ExchangeData struct {
	Data      []map[string]any
	RowNumber int
}

func (ed ExchangeData) NextRowNumber() (nextRowNumber int) {
	return ed.RowNumber + len(ed.Data)
}

type ExcelChanWriter struct {
	fd            *excelize.File
	excelWriter   *_ExcelWriter
	dataChan      chan *ExchangeData
	finishSignal  chan struct{}
	filename      string
	sheet         string
	fieldMetas    FieldMetas
	withTitle     bool
	removeOldFile bool
	err           error
	nextRowNumber int
	streamWriter  *excelize.StreamWriter
	context       context.Context
}

type ExcelChanWriterOption struct {
	WithTitle     bool `json:"withTitle"`
	RemoveOldFile bool `json:"removeOldFile"`
}

func NewExcelChanWriter(ctx context.Context, filename string, sheet string, fieldMetas FieldMetas, option *ExcelChanWriterOption) (ecw *ExcelChanWriter, beginRowNumber int, err error) {
	fieldMetas.Sort()
	if option == nil {
		option = &ExcelChanWriterOption{
			WithTitle:     true,
			RemoveOldFile: true,
		}
	}
	ecw = &ExcelChanWriter{
		excelWriter:   NewExcelWriter(),
		dataChan:      make(chan *ExchangeData, 10),
		finishSignal:  make(chan struct{}, 1),
		filename:      filename,
		sheet:         sheet,
		fieldMetas:    fieldMetas,
		withTitle:     option.WithTitle,
		removeOldFile: option.RemoveOldFile,
		context:       ctx,
	}
	err = ecw.init()
	if err != nil {
		return nil, 0, err
	}
	ecw.subChan()
	return ecw, ecw.nextRowNumber, nil
}

func (ecw *ExcelChanWriter) SendData(exchangeData *ExchangeData) (nextRowNumber int) {
	ecw.dataChan <- exchangeData
	return exchangeData.NextRowNumber()
}

func (ecw *ExcelChanWriter) Finish() (err error) {
	close(ecw.dataChan)
	<-ecw.finishSignal
	if ecw.err != nil {
		return ecw.err
	}
	return nil
}

func (ecw *ExcelChanWriter) init() (err error) {
	excelWriter := ecw.excelWriter
	fd, err := excelWriter.GetFile(ecw.filename, ecw.sheet, ecw.removeOldFile)
	if err != nil {
		return err
	}
	ecw.fd = fd
	streamWriter, nextRowNumber, err := excelWriter.getStreamWriter(fd, ecw.sheet)
	if err != nil {
		return
	}
	if ecw.withTitle {
		chanelData := ExchangeData{
			RowNumber: nextRowNumber,
			Data:      []map[string]any{{}}, // 初始化一个元素
		}
		for _, fieldMeta := range ecw.fieldMetas {
			chanelData.Data[0][fieldMeta.Name] = fieldMeta.Title
		}
		nextRowNumber, err = excelWriter.Write2streamWriter(streamWriter, ecw.fieldMetas, chanelData) //更新nextRowNumber
		if err != nil {
			return err
		}
	}
	ecw.nextRowNumber = nextRowNumber
	ecw.streamWriter = streamWriter
	return nil
}

func (ecw *ExcelChanWriter) subChan() {
	excelWriter := ecw.excelWriter
	streamWriter := ecw.streamWriter
	//开启协程接收数据
	go func() {
		defer func() {
			if result := recover(); result != nil {
				ecw.err = errors.New(fmt.Sprintf("%v+", result))

			}
			// 退出前,先保存数据
			err := ecw.streamWriter.Flush()
			if err != nil {
				ecw.err = err
			}
			err = ecw.fd.Save()
			if err != nil {
				ecw.err = err
			}
			ecw.finishSignal <- struct{}{} // 通知已经完成数据写入
		}()
		for {
			select {
			case <-ecw.context.Done():
				ecw.err = ecw.context.Err()
				return
			case proxyData := <-ecw.dataChan: //读取数据写入文件
				_, err := excelWriter.Write2streamWriter(streamWriter, ecw.fieldMetas, *proxyData)
				if err != nil {
					ecw.err = err
					return
				}
			}
		}
	}()
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
