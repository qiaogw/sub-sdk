package converter

import (
	"fmt"
	"github.com/zeromicro/ddl-parser/parser"
	"reflect"
	"strings"
)

// commonMysqlDataTypeMapInt 存放了 int 型常量对应的 MySQL 数据类型与 Go 基础类型的映射关系。
// 其中 key 来自 parser 包中定义的各种 MySQL 类型常量。
// value 为对应的 Go 数据类型（如 int64、float64、time.Time、bool 等）。
var commonMysqlDataTypeMapInt = map[int]string{
	// 整数类型统一映射成 int64
	parser.Bit:       "byte",
	parser.TinyInt:   "int64",
	parser.SmallInt:  "int64",
	parser.MediumInt: "int64",
	parser.Int:       "int64",
	parser.MiddleInt: "int64",
	parser.Int1:      "int64",
	parser.Int2:      "int64",
	parser.Int3:      "int64",
	parser.Int4:      "int64",
	parser.Int8:      "int64",
	parser.Integer:   "int64",
	parser.BigInt:    "int64",
	// 浮点和定点数字类型统一映射成 float64
	parser.Float:   "float64",
	parser.Float4:  "float64",
	parser.Float8:  "float64",
	parser.Double:  "float64",
	parser.Decimal: "float64",
	parser.Dec:     "float64",
	parser.Fixed:   "float64",
	parser.Numeric: "float64",
	parser.Real:    "float64",
	// 日期/时间类型，有的映射为 time.Time，有的映射为字符串或整数
	parser.Date:      "time.Time",
	parser.DateTime:  "time.Time",
	parser.Timestamp: "time.Time",
	parser.Time:      "string",
	parser.Year:      "int64",
	// 字符串类型统一映射为 string
	parser.Char:            "string",
	parser.VarChar:         "string",
	parser.NVarChar:        "string",
	parser.NChar:           "string",
	parser.Character:       "string",
	parser.LongVarChar:     "string",
	parser.LineString:      "string",
	parser.MultiLineString: "string",
	parser.Binary:          "string",
	parser.VarBinary:       "string",
	parser.TinyText:        "string",
	parser.Text:            "string",
	parser.MediumText:      "string",
	parser.LongText:        "string",
	parser.Enum:            "string",
	parser.Set:             "string",
	parser.Json:            "string",
	parser.Blob:            "string",
	parser.LongBlob:        "string",
	parser.MediumBlob:      "string",
	parser.TinyBlob:        "string",
	// 布尔类型
	parser.Bool:    "bool",
	parser.Boolean: "bool",
}

// commonMysqlDataTypeMapString 存放了字符串类型的 MySQL 字段类型名称（如 "tinyint"、"varchar" 等）
// 与对应 Go 基础类型的映射关系。在处理纯字符串类型的字段类型时使用。
var commonMysqlDataTypeMapString = map[string]string{
	// 布尔类型
	"bool":    "bool",
	"boolean": "bool",
	// 整数类型统一映射成 int64
	"tinyint":   "int64",
	"smallint":  "int64",
	"mediumint": "int64",
	"int":       "int64",
	"int1":      "int64",
	"int2":      "int64",
	"int3":      "int64",
	"int4":      "int64",
	"int8":      "int64",
	"integer":   "int64",
	"bigint":    "int64",
	// 浮点和定点数字类型统一映射成 float64
	"float":   "float64",
	"float4":  "float64",
	"float8":  "float64",
	"double":  "float64",
	"decimal": "float64",
	"dec":     "float64",
	"fixed":   "float64",
	"real":    "float64",
	"bit":     "byte",
	// 日期/时间类型
	"date":      "time.Time",
	"datetime":  "time.Time",
	"timestamp": "time.Time",
	"time":      "string",
	"year":      "int64",
	// 字符串类型
	"linestring":      "string",
	"multilinestring": "string",
	"nvarchar":        "string",
	"nchar":           "string",
	"char":            "string",
	"character":       "string",
	"varchar":         "string",
	"binary":          "string",
	"bytea":           "string",
	"longvarbinary":   "string",
	"varbinary":       "string",
	"tinytext":        "string",
	"text":            "string",
	"mediumtext":      "string",
	"longtext":        "string",
	"enum":            "string",
	"set":             "string",
	"json":            "string",
	"jsonb":           "string",
	"blob":            "string",
	"longblob":        "string",
	"mediumblob":      "string",
	"tinyblob":        "string",
}

// ConvertDataType 根据 parser 中的数据库类型常量（int 值）和是否默认可空 (isDefaultNull)，
// 返回对应的 Go 类型字符串。如果找不到匹配，会返回错误。
// 例如：ConvertDataType(parser.Int, false) => "int64"
func ConvertDataType(dataBaseType int, isDefaultNull bool) (string, error) {
	tp, ok := commonMysqlDataTypeMapInt[dataBaseType]
	if !ok {
		return "", fmt.Errorf("unsupported database type: %v", dataBaseType)
	}
	return mayConvertNullType(tp, isDefaultNull), nil
}

// ConvertStringDataType 根据字符串表示的数据库类型（如 "varchar"、"int" 等）和是否默认可空 (isDefaultNull)，
// 返回对应的 Go 类型字符串。如果找不到匹配，会返回错误。
// 例如：ConvertStringDataType("int", false) => "int64"
func ConvertStringDataType(dataBaseType string, isDefaultNull bool) (string, error) {
	tp, ok := commonMysqlDataTypeMapString[strings.ToLower(dataBaseType)]
	if !ok {
		return "", fmt.Errorf("不支持数据库数据类型: %s", dataBaseType)
	}
	return mayConvertNullType(tp, isDefaultNull), nil
}

// mayConvertNullType 用于根据 isDefaultNull 来决定是否返回 sql.NullXXX 类型。
// 如果 isDefaultNull == true，且 goDataType 属于可转换的基础类型，则返回对应的 sql.NullXXX；
// 否则返回原始 goDataType。
func mayConvertNullType(goDataType string, isDefaultNull bool) string {
	if !isDefaultNull {
		return goDataType
	}

	switch goDataType {
	case "int64":
		return "sql.NullInt64"
	case "int32":
		return "sql.NullInt32"
	case "float64":
		return "sql.NullFloat64"
	case "bool":
		return "sql.NullBool"
	case "string":
		return "sql.NullString"
	case "time.Time":
		return "sql.NullTime"
	default:
		return goDataType
	}
}

// ConvertDefault 用于从一个结构体对象中提取默认值（仅针对特定类型做处理）。
// 如果 c 是一个 sql.NullString，则返回其内部的 String 字段。
// 否则，返回空字符串。
func ConvertDefault(c interface{}) string {
	cType := reflect.TypeOf(c)
	cValue := reflect.ValueOf(c)
	// 如果类型名是 "NullString"，则返回其 String 字段的值
	if cType.Name() == "NullString" {
		return cValue.FieldByName("String").String()
	}
	// 否则返回空串
	return ""
}
