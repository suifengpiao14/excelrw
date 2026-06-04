# excelrw

Go Excel 读写库，基于 [excelize](https://github.com/qax-os/excelize) 封装，提供流式写入（大数据量导出）、读取、HTTP 代理分页导出、数据库配置化导出、JavaScript 动态脚本等能力。

## 安装

```bash
go get github.com/suifengpiao14/excelrw
```

要求 Go >= 1.23。

## 快速开始

### 流式写入 Excel（推荐）

适用于大数据量导出场景，通过分页回调逐批写入，内存占用低。

```go
ctx := context.Background()

// 定义列映射：Name 为数据字段名，Title 为 Excel 列标题
fieldMetas := defined.FieldMetas{
    {Name: "__rowNumber", Title: "序号"},
    {Name: "name", Title: "姓名"},
    {Name: "age", Title: "年龄"},
    {Name: "{{city}}({{district}})", Title: "城市(区)"}, // 支持模板组合多字段
}

ecw := excelrw.NewExcelStreamWriter(ctx, "./output/example.xlsx").
    WithFieldMetas(fieldMetas).
    WithFetcher(func(loopCount int) (rows []map[string]string, err error) {
        // 第 N 次调用返回第 N 页数据，返回空切片表示结束
        if loopCount > 1 {
            return nil, nil
        }
        return []map[string]string{
            {"name": "张三", "age": "28", "city": "深圳", "district": "南山"},
            {"name": "李四", "age": "32", "city": "北京", "district": "朝阳"},
        }, nil
    })

errChan, err := ecw.Run()
if err != nil {
    log.Fatal(err)
}
if err := <-errChan; err != nil {
    log.Fatal(err)
}
```

### 读取 Excel

```go
f, err := excelize.OpenFile("./example.xlsx")
if err != nil {
    log.Fatal(err)
}

reader := excelrw.NewExcelReader()
// fieldMap: Excel 列名(A,B,C...) → 字段名映射；rowIndex=2 从第 2 行开始读
data, err := reader.Read(f, "sheet1", map[string]string{
    "a": "Fsort",
    "b": "Ftype",
    "c": "Funique_code",
}, 2, true) // isUnmergeCell=true 自动展开合并单元格
```

### 数据类型转换

将 `[]struct{}` 或 `[]map[string]any` 转为写入所需的 `[]map[string]string`：

```go
type User struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

users := []*User{{"张三", 18}, {"李四", 20}}
rows := excelrw.SliceAny2stringMust(users)

// 也可以从 []map[string]any 转换
rows = excelrw.SliceAny2stringMust([]map[string]any{
    {"name": "张三", "age": 18},
})
```

### 简单导出（自定义数据获取）

```go
export := excelrw.NewExportExcel("output.xlsx", fieldMetas).
    WithFetcherDataFn(func(loopCount int, param map[string]any) (rows []map[string]string, err error) {
        // 根据 param 查询数据库或调用 API，分页返回数据
        return myQuery(param, loopCount)
    })

filename, err := export.Export(map[string]any{
    "userId": 123,
    "status": "active",
})
```

### HTTP 代理分页导出（ExportApi）

自动代理 HTTP 请求、自动分页翻页、流式写入 Excel。适合导出已有列表接口数据的场景。

```go
in := excelrw.ExportApiIn{
    ProxyRquest: excelrw.ProxyRquest{
        RequestDTO: httpraw.RequestDTO{
            Method:  "POST",
            URL:     "https://api.example.com/list",
            Body:    `{"pageIndex":1,"pageSize":100}`,
            Headers: map[string]string{"Content-Type": "application/json"},
        },
        PageIndexPath:  "pageIndex",   // body 中页码字段路径
        PageIndexStart: "1",           // 起始页码
        PageSizePath:   "pageSize",    // body 中每页大小字段路径
        PageSize:       200,           // 覆盖每页大小
    },
    ProxyResponse: excelrw.ProxyResponse{
        DataPath:         "data.list", // 响应中数据列表路径
        BusinessCodePath: "code",      // 业务状态码路径
        BusinessOkCode:   "0",         // 成功状态码值
    },
    Settings: excelrw.Settings{
        Filename:   "./output/export.xlsx",
        FieldMetas: fieldMetas,
    },
}

errChan, err := excelrw.ExportApi(in)
if err != nil {
    log.Fatal(err)
}
if err := <-errChan; err != nil {
    log.Fatal(err)
}
```

### 数据库配置化导出（MakeExportApiIn）

通过数据库存储导出配置（请求模板、字段映射、动态脚本等），运行时动态渲染导出任务。

```go
// 从数据库获取导出配置
config, err := configRepo.GetMust(repository.ExportConfigRepositoryGetIn{
    ConfigKey: "order_export",
})

// 构建导出参数并执行
exportApiIn, err := excelrw.MakeExportApiIn(excelrw.MakeExportApiInArgs{
    ConfigKey: "order_export",
    CreatorId: "123",
    Filename:  "/static/export/orders.xlsx",
    Request: excelrw.Request{
        Body:    json.RawMessage(`{"status":"active"}`),
        Headers: map[string]string{"Authorization": "Bearer xxx"},
    },
}, config)

errChan, err := excelrw.ExportApi(exportApiIn)
```

### JavaScript 动态脚本

在导出配置中嵌入 JS 脚本，用于请求格式化、记录转换、动态设置等。基于 [goja](https://github.com/dop251/goja) 引擎。

```javascript
// 请求格式化：修改发送给数据源的请求
function requestFormatFn(requestDTO) {
    var body = JSON.parse(requestDTO.body);
    body.extraField = "value";
    requestDTO.body = JSON.stringify(body);
    return requestDTO;
}

// 记录格式化：转换每行数据
function recordFormatFn(record) {
    record.displayName = record.lastName + record.firstName;
    return record;
}

// 动态设置：根据请求体动态修改文件名和列定义
function settingFn(body) {
    var parsed = JSON.parse(body);
    return {
        filename: "/static/export/" + parsed.type + ".xlsx",
        titles: [
            { name: "id", title: "ID" },
            { name: "name", title: "名称" }
        ]
    };
}
```

JS 环境内置工具函数：`md5(str)`、`base64Encode(str)`、`base64Decode(str)`、`console.log(...)`。

## 核心概念

### FieldMeta（字段映射）

| 字段 | 说明 |
|------|------|
| `Name` | 数据字段名，支持 mustache 模板语法组合多字段，如 `{{city}}({{district}})` |
| `Title` | Excel 列标题 |

内置特殊字段：
- `__rowNumber` — 自动生成行号（从 0 开始）

列宽会根据第一页数据内容自动计算调整。

### ExcelStreamWriter（流式写入器）

| 方法 | 说明 |
|------|------|
| `WithFieldMetas(metas)` | 设置字段映射 |
| `WithFetcher(fn)` | 设置数据获取回调 |
| `WithInterval(d)` | 设置分页请求间隔 |
| `WithDeleteFile(d, handler)` | 延迟自动删除导出文件 |
| `WithMaxLoopCount(n)` | 最大循环次数（默认 1,000,000） |
| `WithSheet(name)` | 指定 Sheet 名称（默认 `sheet1`） |
| `WithAppendToExistsFile()` | 追加到已存在的文件 |
| `Run()` | 启动异步导出，返回 `chan error` |

### ExportApi（HTTP 代理导出）

通过代理已有列表接口实现分页导出，主要配置：

| 配置 | 说明 |
|------|------|
| `ProxyRquest` | 代理请求参数（URL、Method、Body、分页路径等） |
| `ProxyResponse` | 响应解析参数（数据路径、业务状态码路径等） |
| `Settings` | 导出设置（文件名、字段映射、间隔等） |

## 子包

### `defined`

定义核心类型：`FieldMeta`、`FieldMetas`、`RecordFormatFn`、`RequestFormatFn`、`SettingFn` 等。

### `repository`

数据库模型和仓储层，支持导出配置管理、导出任务管理、回调配置、请求日志记录。

| 模型 | 说明 |
|------|------|
| `ExportConfigModel` | 导出配置（请求模板、字段映射、动态脚本等） |
| `ExportTaskModel` | 导出任务（状态跟踪：exporting/success/fail） |
| `ExportCallbackConfig` | 导出完成回调配置 |
| `RequestLog` | 请求日志记录 |

支持通过 `sqlbuilder.TableConfig` 自动生成 DDL。

### `dynamichook`

动态脚本引擎，支持两种模式：
- **JavaScript**（推荐）：基于 goja，提供 `JSVM` 及 `RecordFormatFn`、`RequestFormatFn`、`SettingFn` 等 JS 函数绑定
- **Go 动态编译**：基于 yaegi，用于请求/响应中间件

## License

MIT
