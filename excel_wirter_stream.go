package excelrw

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/suifengpiao14/excelrw/defined"
	"github.com/xuri/excelize/v2"
)

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

func (excelWriter *_ExcelWriter) SetColWidth(streamWriter *excelize.StreamWriter, fieldMetas defined.FieldMetas) (err error) {
	colLen := len(fieldMetas)
	for i := range colLen {
		fieldMeta := fieldMetas[i]
		maxSize := fieldMeta.GetMaxSize()
		if maxSize > 0 {
			col := i + 1
			err = streamWriter.SetColWidth(col, col, float64(maxSize)) // 设置列宽

			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Write2streamWriter 向写入流中写入数据
func (excelWriter *_ExcelWriter) Write2streamWriter(streamWriter *excelize.StreamWriter, fieldMetas defined.FieldMetas, rowNumber int, rows []map[string]string) (nextRowNumber int, err error) {
	colLen := len(fieldMetas)
	minColIndex := 1

	for _, record := range rows {
		// 组装一行数据
		row := make([]any, colLen)
		for i := 0; i < colLen; i++ {
			row[i] = record[fieldMetas[i].Name]
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

type FetcherFn func(loopCount int) (rows []map[string]string, forceBreak bool, err error)

type ExcelStreamWriter struct {
	fd                *excelize.File
	excelWriter       *_ExcelWriter
	filename          string
	sheet             string
	fieldMetas        defined.FieldMetas
	withoutTitleRow   bool
	RemoveFileTimeout time.Duration

	nextRowNumber int
	streamWriter  *excelize.StreamWriter
	context       context.Context
	fetcher       FetcherFn
	interval      time.Duration
	maxLoopCount  int // 最大循环次数
}

func NewExcelStreamWriter(ctx context.Context, filename string, fieldMetas defined.FieldMetas) (ecw *ExcelStreamWriter) {
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
func (ecw *ExcelStreamWriter) WithFieldMetas(fieldMetas defined.FieldMetas) *ExcelStreamWriter {
	ecw.fieldMetas = fieldMetas
	return ecw
}

func (ecw ExcelStreamWriter) GetFilename() string {
	return ecw.filename
}

func (ecw *ExcelStreamWriter) WithoutTitleRow() *ExcelStreamWriter {
	ecw.withoutTitleRow = true
	return ecw
}

func (ecw *ExcelStreamWriter) AutoAdjustColumnWidth() (err error) {
	for i, fieldMeta := range ecw.fieldMetas {
		columnNumber := i + 1
		col, _ := excelize.ColumnNumberToName(columnNumber)
		colMax, _ := excelize.ColumnNumberToName(columnNumber + 1)
		maxSize := fieldMeta.GetMaxSize()                                  // 测试使用
		err = ecw.fd.SetColWidth(ecw.sheet, col, colMax, float64(maxSize)) // 乘以256，因为excel的列宽是以1/256个字符宽度为单位的。
		if err != nil {
			return err
		}
	}
	return nil
}

// CalFieldMetaMaxSize 计算字段最大长度，用于自动调整列宽
func (ecw *ExcelStreamWriter) CalFieldMetaMaxSize(rows []map[string]string) {
	for i := 0; i < len(ecw.fieldMetas); i++ {
		key := ecw.fieldMetas[i].Name
		for _, record := range rows {
			content := record[key]
			lineIndex := strings.Index(content, "\n")
			if lineIndex > 0 {
				content = content[:lineIndex]
			}
			maxSize := len(content)
			if isNumber(content) {
				maxSize += 3 // 数字(如身份证)额外增加3个字符宽度，以便于显示美观

			}
			ecw.fieldMetas[i].SetMaxSize(maxSize)
		}
	}
}

func isNumber(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func (ecw *ExcelStreamWriter) WithMaxLoopCount(maxLoopCount int) *ExcelStreamWriter {
	ecw.maxLoopCount = maxLoopCount
	return ecw
}

var DefalutMaxLoopCountLimit = 1000000 //最多循环次数限制

func (ecw *ExcelStreamWriter) gethMaxLoopCount() int {
	if ecw.maxLoopCount > 0 {
		return ecw.maxLoopCount
	}

	return DefalutMaxLoopCountLimit
}

func (ecw *ExcelStreamWriter) WithDeleteFile(delay time.Duration, errorHandler func(err error)) *ExcelStreamWriter {
	if delay <= 0 {
		return ecw
	}
	if errorHandler == nil {
		errorHandler = func(err error) {
			fmt.Println("ExcelStreamWriter.WithDeleteFile error", err)
		}
	}
	go func() {
		// 等待指定时间
		time.Sleep(delay)
		// 删除文件
		err := os.Remove(ecw.filename)
		if err != nil {
			errorHandler(err)
		}
	}()
	return ecw
}

// WithFetcher 设置数据获取器，用于从数据库或其他地方获取数据并写入Excel文件,可以使用SliceAny2string辅助函数 在回调FetcherFn 中转换数据类型输出
func (ecw *ExcelStreamWriter) WithFetcher(fetcher FetcherFn) *ExcelStreamWriter {
	ecw.fetcher = fetcher
	return ecw
}
func (ecw *ExcelStreamWriter) WithInterval(interval time.Duration) *ExcelStreamWriter {
	ecw.interval = interval
	return ecw
}

func (ecw *ExcelStreamWriter) GetFiledMetas() (fields defined.FieldMetas, err error) {
	if ecw.fieldMetas == nil {
		return fields, errors.New("fieldMetas is nil")
	}
	return ecw.fieldMetas, nil
}

func (ecw *ExcelStreamWriter) init() (err error) {

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

// Run 执行导出 ,返回错误通道,如果需要同步，则调用方只需同步等待errChan结果即可，若为异步执行，则调用方只需将errChan异步处理即可或者忽略
func (ecw *ExcelStreamWriter) Run() (errChan chan error, err error) {

	err = ecw.init()
	if err != nil {
		return nil, err
	}
	errChan = make(chan error)
	go func() {
		err := ecw.loop()
		errChan <- err
		close(errChan)
	}()

	return errChan, nil

}
func (ecw *ExcelStreamWriter) loop() (err error) {
	loopCount := 0
	maxLoopCount := ecw.gethMaxLoopCount()
	defer ecw.save()
	for {
		select {
		case <-ecw.context.Done():
			return ecw.context.Err()
		default:
		}
		if loopCount > maxLoopCount {
			err = errors.Errorf("loop times is over limit:%d", maxLoopCount)
			return err
		}
		loopCount++
		data, forceBreak, err := ecw.fetcher(loopCount)
		if err != nil {
			return err
		}
		if loopCount == 1 { // 第一次循环 ,写在len(data) == 0之前,确保需要写入标题时，一定会写入标题行数据,方便调试和测试)
			if !ecw.withoutTitleRow { // 第一次循环，增加标题行数据
				data = append([]map[string]string{ecw.getTitleRow()}, data...) //添加到第一行
			}
			// 使用第一次数据作为样本(包含标题和实际数据),计算最大列宽
			ecw.CalFieldMetaMaxSize(data)
			// 设置列宽(必须在写入数据之前调用)
			err = ecw.setColWidth()
			if err != nil {
				return err
			}
		}

		if len(data) == 0 {
			break
		}

		ecw.nextRowNumber, err = ecw.writeData(ecw.nextRowNumber, data)
		if err != nil {
			return err
		}
		if forceBreak {
			break
		}
		if ecw.interval > 0 {
			time.Sleep(ecw.interval)
		}
	}

	return nil
}

func (ecw *ExcelStreamWriter) getTitleRow() (row map[string]string) {
	row = ecw.fieldMetas.MakeTitleRow()
	return row
}
func (ecw *ExcelStreamWriter) setColWidth() (err error) {
	err = ecw.excelWriter.SetColWidth(ecw.streamWriter, ecw.fieldMetas) // 设置列宽(必须在写入数据之前调用)
	if err != nil {
		return err
	}
	return nil
}

func (ecw *ExcelStreamWriter) writeData(rowNumber int, rows []map[string]string) (nextRowNumber int, err error) {
	fieldMetas, err := ecw.GetFiledMetas()
	if err != nil {
		return 0, err
	}
	nextRowNumber, err = ecw.excelWriter.Write2streamWriter(ecw.streamWriter, fieldMetas, rowNumber, rows)
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
