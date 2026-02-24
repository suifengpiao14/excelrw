package defined

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hoisie/mustache"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/suifengpiao14/httpraw"
)

type FieldMeta struct {
	Title    string `json:"title"` // 列标题
	Name     string `json:"name"`  // 列值模板，例如：{{nameField}}({{idField}}),如果只有一个字段，则可以省略{{}}
	maxSize  int    // 当前列字符串最多的个数(用来调整列宽)
	template *mustache.Template
	err      error
}

var ErrorFieldMeta = errors.Errorf("FieldMeta.Name is empty")

func (fm *FieldMeta) parseTpl() (*mustache.Template, error) {
	if fm.template != nil {
		return fm.template, nil
	}
	if fm.Name == "" {
		fm.err = ErrorFieldMeta
		return nil, fm.err
	}
	tpl := fm.Name
	if !strings.Contains(fm.Name, "{{") {
		tpl = fmt.Sprintf(`{{%s}}`, fm.Name)
	}

	fm.template, fm.err = mustache.ParseString(tpl)
	return fm.template, fm.err
}

func (fm FieldMeta) GetValue(rowNumber int, row map[string]string) string {
	if fm.err != nil {
		return fm.err.Error()
	}
	if value, ok := row[fm.Name]; ok {
		return value
	}
	m := map[string]any{"__rowNumber": rowNumber}
	if value, ok := m[fm.Name]; ok {
		return cast.ToString(value)
	}
	tpl, err := fm.parseTpl()
	if err != nil {
		return err.Error()
	}
	value := tpl.Render(row, m)
	return value
}
func (fm FieldMeta) GetMaxSize() int { return fm.maxSize }

var ColumnMaxSize = 100 // 列宽最大值

func (fm *FieldMeta) SetMaxSize(size int) {
	if size > ColumnMaxSize {
		size = ColumnMaxSize // 列宽最大值限制

	}
	if fm.maxSize < size {
		fm.maxSize = size
	}
}

type FieldMetas []FieldMeta

func (fs FieldMetas) MakeTitleRow() map[string]string {
	m := make(map[string]string)
	for _, fieldMeta := range fs {
		m[fieldMeta.Name] = fieldMeta.Title
	}
	return m

}
func (fs FieldMetas) Empty() bool {
	return len(fs) == 0
}

func (fs *FieldMetas) Unmarshal(fieldMetasStr string) (err error) {
	if fieldMetasStr == "" {
		return nil
	}
	return json.Unmarshal([]byte(fieldMetasStr), fs)
}

type RecordFormatFn func(record map[string]string) (newRecord map[string]string, err error)
type RequestFormatFn func(requestDTO httpraw.RequestDTO) (newRequestDTO httpraw.RequestDTO, err error)
type ResponseFormatFn func(responseDTO httpraw.ResponseDTO) (records []map[string]any, err error)
type Setting struct {
	Filename string     `json:"filename"`
	Titles   FieldMetas `json:"titles"`
}
type SettingFn func(body string) (Setting Setting, err error)
