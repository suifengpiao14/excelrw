package excelrw

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

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

func (fs FieldMetas) MakeTitleRow() map[string]string {
	m := make(map[string]string)
	for _, fieldMeta := range fs {
		m[fieldMeta.Name] = fieldMeta.Title
	}
	return m

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

type _ExcelWriter struct{}

func NewExcelWriter() (writer *_ExcelWriter) {
	return &_ExcelWriter{}
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

// Write2streamWriter 向写入流中写入数据
func (excelWriter *_ExcelWriter) Write2streamWriter(streamWriter *excelize.StreamWriter, fieldMetas FieldMetas, rowNumber int, rows []map[string]string) (nextRowNumber int, err error) {
	fieldMetas.Sort()
	colLen := fieldMetas.Len()
	minColIndex := fieldMetas.MinColIndex() // 找到最小的列
	for _, record := range rows {
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

// GetStreamWriter 打开文件流，将已有的数据填写到流内，返回写入流
func (excelWriter *_ExcelWriter) GetStreamWriter(fd *excelize.File, sheet string) (streamWriter *excelize.StreamWriter, nextRowNumber int, err error) {
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

type ExcelStreamWriter struct {
	fd                *excelize.File
	excelWriter       *_ExcelWriter
	filename          string
	sheet             string
	async             bool
	fieldMetas        FieldMetas
	withTitleRow      bool
	RemoveFileTimeout time.Duration
	nextRowNumber     int
	streamWriter      *excelize.StreamWriter
	context           context.Context
	fetcher           func(loopIndex int) (rows []map[string]string, err error)
	asyncErrorHandler func(err error)
}

func NewExcelStreamWriter(ctx context.Context, filename string, fieldMetas FieldMetas) (ecw *ExcelStreamWriter) {
	fieldMetas.Sort()
	excelWriter := NewExcelWriter()
	ecw = &ExcelStreamWriter{
		excelWriter: excelWriter,
		filename:    filename,
		sheet:       SheetDefault,
		fieldMetas:  fieldMetas,
		context:     ctx,
	}
	return ecw
}

func (ecw *ExcelStreamWriter) WithSheet(sheet string) *ExcelStreamWriter {
	ecw.sheet = sheet
	return ecw
}
func (ecw *ExcelStreamWriter) WithAsync() *ExcelStreamWriter {
	ecw.async = true
	return ecw
}

func (ecw *ExcelStreamWriter) WithTitleRow() *ExcelStreamWriter {
	ecw.withTitleRow = true
	return ecw
}

func (ecw *ExcelStreamWriter) WithDeleteFile(delay time.Duration) *ExcelStreamWriter {
	go func() {
		// 等待指定时间
		time.Sleep(delay)
		// 删除文件
		err := os.Remove(ecw.filename)
		if err != nil {
			ecw.getAsyncErrorHandler()(err)
		}
	}()
	return ecw
}

func (ecw *ExcelStreamWriter) WithFetcher(fetcher func(loopIndex int) (rows []map[string]string, err error)) *ExcelStreamWriter {
	ecw.fetcher = fetcher
	return ecw
}

func (ecw *ExcelStreamWriter) WithAsyncErrorHandler(asyncErrorHandler func(err error)) *ExcelStreamWriter {
	ecw.asyncErrorHandler = asyncErrorHandler
	return ecw
}

func (ecw *ExcelStreamWriter) getAsyncErrorHandler() func(err error) {
	if ecw.asyncErrorHandler == nil {
		ecw.asyncErrorHandler = func(err error) {
			fmt.Println(" ExcelStreamWriter error:", err)
		}
	}
	return ecw.asyncErrorHandler
}

func (ecw *ExcelStreamWriter) init() (err error) {
	if ecw.fieldMetas == nil {
		return errors.New("fieldMetas is nil")
	}
	if ecw.fetcher == nil {
		return errors.New("fetcher is nil")
	}

	fd, err := ecw.excelWriter.GetFile(ecw.filename, ecw.sheet, true)
	if err != nil {
		return err
	}
	ecw.fd = fd
	streamWriter, nextRowNumber, err := ecw.excelWriter.GetStreamWriter(fd, ecw.sheet)
	if err != nil {
		return
	}

	ecw.nextRowNumber = nextRowNumber
	ecw.streamWriter = streamWriter

	return
}

func (ecw *ExcelStreamWriter) Run() (err error) {

	err = ecw.init()
	if err != nil {
		return err
	}
	if ecw.withTitleRow {
		err = ecw.addTitleRow()
		if err != nil {
			return err
		}
	}
	if ecw.async {
		go ecw.loop()
	} else {
		err = ecw.loop()
		if err != nil {
			return err
		}
	}

	return nil

}
func (ecw *ExcelStreamWriter) loop() (err error) {
	loopIndex := 0
	for {
		select {
		case <-ecw.context.Done():
			return ecw.context.Err()
		default:
		}
		data, err := ecw.fetcher(loopIndex)
		if err != nil {
			return err
		}
		if len(data) == 0 {
			break
		}
		loopIndex++
		ecw.nextRowNumber, err = ecw.writeData(ecw.nextRowNumber, data)
		if err != nil {
			return err
		}
	}
	err = ecw.save()
	if err != nil {
		return err
	}
	return nil
}

func (ecw *ExcelStreamWriter) addTitleRow() (err error) {
	row := ecw.fieldMetas.MakeTitleRow()
	rows := []map[string]string{row}
	ecw.nextRowNumber, err = ecw.excelWriter.Write2streamWriter(ecw.streamWriter, ecw.fieldMetas, ecw.nextRowNumber, rows) //更新nextRowNumber
	if err != nil {
		return err
	}
	return nil
}

func (ecw *ExcelStreamWriter) writeData(rowNumber int, rows []map[string]string) (nextRowNumber int, err error) {
	nextRowNumber, err = ecw.excelWriter.Write2streamWriter(ecw.streamWriter, ecw.fieldMetas, rowNumber, rows)
	if err != nil {
		return 0, err
	}
	return nextRowNumber, err
}

func (ecw *ExcelStreamWriter) save() (err error) {
	err = ecw.streamWriter.Flush()
	if err != nil {
		return err
	}
	err = ecw.fd.Save()
	if err != nil {
		return err
	}
	err = ecw.fd.Close()
	if err != nil {
		return err
	}

	return nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// 延迟删除文件