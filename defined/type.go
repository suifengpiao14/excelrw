package defined

import (
	"encoding/json"
	"strings"

	"github.com/hoisie/mustache"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/suifengpiao14/httpraw"
	"github.com/xuri/excelize/v2"
)

func RenderFnName(name string) Render {
	return RenderFn(func(context ...any) string {
		return name
	})
}

type Render interface {
	Render(context ...any) string
}

type RenderFn func(context ...any) string

func (r RenderFn) Render(context ...any) string {
	return r(context...)
}

type FieldMeta struct {
	Title   string `json:"title"` // 列标题
	Name    string `json:"name"`  // 列值模板，例如：{{nameField}}({{idField}}),如果只有一个字段，则可以省略{{}}
	maxSize int    // 当前列字符串最多的个数(用来调整列宽)
	Render  Render `json:"-"` // 列值渲染器，例如：{{nameField}}({{idField}})
	//template *mustache.Template
	err error
}

func (fm *FieldMeta) getRender() (r Render, err error) {
	if fm.Render != nil {
		return fm.Render, nil
	}
	if fm.Name == "" {
		fm.err = ErrorFieldMeta
		return nil, fm.err
	}
	tpl := fm.Name
	if !strings.Contains(fm.Name, "{{") { // 不包含{{}},直接返回字段名作为值（支持增加常量字段）
		return RenderFnName(fm.Name), nil
	}
	fm.Render, fm.err = mustache.ParseString(tpl)
	return fm.Render, fm.err
}

var ErrorFieldMeta = errors.Errorf("FieldMeta.Name is empty")

func (fm *FieldMeta) GetValue(rowNumber int, row map[string]string) string {
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
	render, err := fm.getRender()
	if err != nil {
		return err.Error()
	}
	value := render.Render(row, m)
	return value
}
func (fm FieldMeta) GetMaxSize() int { return fm.maxSize }

func (fm *FieldMeta) SetMaxSize(size int) {
	if size > excelize.MaxColumnWidth {
		size = excelize.MaxColumnWidth // 列宽最大值限制

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
