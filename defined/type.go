package defined

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hoisie/mustache"
	"github.com/spf13/cast"
)

type FieldMeta struct {
	Title    string `json:"title"`    // 列标题
	ValueTpl string `json:"valueTpl"` // 列值模板，例如：{{nameField}}({{idField}}),如果只有一个字段，则可以省略{{}}
	maxSize  int    // 当前列字符串最多的个数(用来调整列宽)
	template *mustache.Template
	err      error
}

func (fm *FieldMeta) parseTpl() *mustache.Template {
	if fm.template != nil {
		return fm.template
	}
	tpl := fm.ValueTpl
	if !strings.Contains(fm.ValueTpl, "{{") {
		tpl = fmt.Sprintf(`{{%s}}`, fm.ValueTpl)
	}

	fm.template, fm.err = mustache.ParseString(tpl)
	return fm.template
}

func (fm FieldMeta) GetValue(rowNumber int, row map[string]string) string {
	if fm.err != nil {
		return fm.err.Error()
	}
	if value, ok := row[fm.ValueTpl]; ok {
		return value
	}
	m := map[string]any{"__rowNumber": rowNumber}
	if value, ok := m[fm.ValueTpl]; ok {
		return cast.ToString(value)
	}
	value := fm.parseTpl().Render(row, m)
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
		m[fieldMeta.ValueTpl] = fieldMeta.Title
	}
	return m

}

func (fs *FieldMetas) Unmarshal(fieldMetasStr string) (err error) {
	if fieldMetasStr == "" {
		return nil
	}
	return json.Unmarshal([]byte(fieldMetasStr), fs)
}
