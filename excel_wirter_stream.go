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
	_ColNumber int    //`json:"colNumber"` // 列号(数字,1开始) 增加json tag,可方便调用方验证输入是否符合格式 2025-3-11 内部使用的字段，暴露出去后接入方不会有疑惑，因此先改成私有字段。 调用方验证的方便性，通过增加GetColNumber方法代替。
	Name       string `json:"name"`  // 列名称
	Title      string `json:"title"` // 列标题
}

func (fm FieldMeta) GetColNumber() int { return fm._ColNumber }

type FieldMetas []FieldMeta

func (a FieldMetas) Len() int           { return len(a) }
func (a FieldMetas) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a FieldMetas) Less(i, j int) bool { return a[i]._ColNumber < a[j]._ColNumber }

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
		if fieldMeta._ColNumber < 1 {
			(*fs)[i]._ColNumber = i + 1
		}
	}
}

// 最小列
func (fs FieldMetas) MinColIndex() (minColIndex int) {
	sort.Sort(fs)
	if len(fs) > 0 {
		return fs[0]._ColNumber
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
			k := fieldMeta._ColNumber - 1
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

type FetcherFn func(prevPageIndex int) (currentPageIndex int, rows []map[string]string, err error)

type ExcelStreamWriter struct {
	fd                *excelize.File
	excelWriter       *_ExcelWriter
	filename          string
	sheet             string
	fieldMetas        FieldMetas
	withoutTitleRow   bool
	RemoveFileTimeout time.Duration
	nextRowNumber     int
	streamWriter      *excelize.StreamWriter
	context           context.Context
	fetcher           FetcherFn
	interval          time.Duration
	maxLoopCount      int // 最大循环次数
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
func (ecw ExcelStreamWriter) GetFilename() string {
	return ecw.filename
}

func (ecw *ExcelStreamWriter) WithoutTitleRow() *ExcelStreamWriter {
	ecw.withoutTitleRow = true
	return ecw
}
func (ecw *ExcelStreamWriter) WithMaxLoopCount(maxLoopCount int) *ExcelStreamWriter {
	ecw.maxLoopCount = maxLoopCount
	return ecw
}

var DefalutMaxLoopCountLimit = 1000000 //最多循环次数限制

func (ecw *ExcelStreamWriter) githMaxLoopCount() int {
	if ecw.maxLoopCount > 0 {
		return ecw.maxLoopCount
	}

	return DefalutMaxLoopCountLimit
}

func (ecw *ExcelStreamWriter) WithDeleteFile(delay time.Duration, errorHandler func(err error)) *ExcelStreamWriter {
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

// Run 执行导出 ,返回错误通道,如果需要同步，则调用方只需同步等待errChan结果即可，若为异步执行，则调用方只需将errChan异步处理即可或者忽略
func (ecw *ExcelStreamWriter) Run() (errChan chan error, err error) {

	err = ecw.init()
	if err != nil {
		return nil, err
	}
	if !ecw.withoutTitleRow { // 默认写入标题行，除非明确标记不写入标题
		err = ecw.addTitleRow()
		if err != nil {
			return nil, err
		}
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
	loopIndex := 0
	maxLoopCount := ecw.githMaxLoopCount()
	prevPageIndex := -1
	defer ecw.save()
	for {
		select {
		case <-ecw.context.Done():
			return ecw.context.Err()
		default:
		}

		if loopIndex > maxLoopCount {
			err = errors.Errorf("loop times is over limit:%d", maxLoopCount)
			return err
		}

		currentPageIndex, data, err := ecw.fetcher(prevPageIndex)
		if err != nil {
			return err
		}

		if currentPageIndex <= prevPageIndex {
			err = errors.New("pageNumber is not increase")
			return err
		}
		prevPageIndex = currentPageIndex

		if len(data) == 0 {
			break
		}
		loopIndex++
		ecw.nextRowNumber, err = ecw.writeData(ecw.nextRowNumber, data)
		if err != nil {
			return err
		}
		if ecw.interval > 0 {
			time.Sleep(ecw.interval)
		}
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
