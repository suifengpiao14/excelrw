package excelrw

import (
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

//OpenReader 打开excel 文件(通过post过来的临时文件句柄打开文件)
func OpenReader(r io.Reader) (f *excelize.File, err error) {
	f, err = excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}
	return f, err
}

type _ExcelReader struct {
}

// NewExcelReader 实例化 excel reader服务
func NewExcelReader() *_ExcelReader {
	return &_ExcelReader{}
}

// Read 读取excel 表中所有数据 fieldMap key 为 a、b、c等excel列名称,value为这列转换为记录的属性名
func (instance *_ExcelReader) Read(f *excelize.File, sheet string, fieldMap map[string]string, rowIndex int, isUnmergeCell bool) ([]map[string]string, error) {
	if isUnmergeCell {
		err := instance.UnmergeCell(f, sheet)
		if err != nil {
			return nil, err
		}
	}
	// 获取 Sheet 上所有单元格
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	output := make([]map[string]string, 0)
	for k, v := range fieldMap {
		fieldMap[strings.ToUpper(k)] = v // 兼容 大小写
	}

	for index, row := range rows {
		if index < rowIndex-1 { // 从指定行开始读取
			continue
		}
		record := make(map[string]string, 0)
		for colIndex, colCell := range row {
			colName, err := excelize.ColumnNumberToName(colIndex + 1)
			if err != nil {
				return nil, err
			}
			if fieldMap != nil { // 如果定制了列名和字段映射关系，替换字段映射
				field, ok := fieldMap[colName]
				if ok {
					record[field] = colCell
				}
			} else {
				record[colName] = colCell
			}
		}
		output = append(output, record)
	}
	return output, nil
}

//UnmergeCell 将合并单元格展开，值填充到每个展开的单元内
func (instance *_ExcelReader) UnmergeCell(f *excelize.File, sheet string) (err error) {
	mergeCells, err := f.GetMergeCells(sheet)
	if err != nil {
		return err
	}
	if len(mergeCells) == 0 {
		return nil
	}
	for _, mergeCell := range mergeCells {
		startAxis := mergeCell.GetStartAxis()
		endAxis := mergeCell.GetEndAxis()
		value := mergeCell.GetCellValue()
		err = f.UnmergeCell(sheet, startAxis, endAxis)
		if err != nil {
			return err
		}
		cells, err := expandCellRegion(startAxis, endAxis)
		if err != nil {
			return err
		}
		for _, cell := range cells {
			err = f.SetCellValue(sheet, cell, value)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

// expandCellRegion 将合并区域展开成单个单元地址集合
func expandCellRegion(startAxis string, endAxis string) (cells []string, err error) {
	startColumn, startRow, err := excelize.CellNameToCoordinates(startAxis)
	if err != nil {
		return nil, err
	}
	endColumn, endRow, err := excelize.CellNameToCoordinates(endAxis)
	if err != nil {
		return nil, err
	}

	cells = make([]string, 0)
	for row := startRow; row <= endRow; row++ {
		for column := startColumn; column <= endColumn; column++ {
			cell, err := excelize.CoordinatesToCellName(column, row)
			if err != nil {
				return nil, err
			}
			cells = append(cells, cell)
		}
	}
	return cells, nil
}
