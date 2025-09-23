package defined

import "encoding/json"

type FieldMeta struct {
	Name    string `json:"name"`  // 列名称
	Title   string `json:"title"` // 列标题
	maxSize int    // 当前列字符串最多的个数(用来调整列宽)
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

func (fs *FieldMetas) Unmarshal(fieldMetasStr string) (err error) {
	if fieldMetasStr == "" {
		return nil
	}
	return json.Unmarshal([]byte(fieldMetasStr), fs)
}
