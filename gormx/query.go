package gormx

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	FromQueryTag = "search"   // FromQueryTag tag标记
	Mysql        = "mysql"    // Mysql 数据库标识
	Postgres     = "postgres" // Postgres 数据库标识
)
const (
	Exact       = "exact"       // 精确匹配，相当于 SQL 的 `=`
	IExact      = "iexact"      // 不区分大小写的精确匹配（PostgreSQL 适用）
	Contains    = "contains"    // 包含匹配，相当于 SQL 的 `LIKE %xxx%`
	IContains   = "icontains"   // 不区分大小写的包含匹配（PostgreSQL `ILIKE %xxx%`）
	Greater     = "gt"          // 大于 (`>`)，SQL: `column > value`
	GreaterEq   = "gte"         // 大于等于 (`>=`)，SQL: `column >= value`
	Less        = "lt"          // 小于 (`<`)，SQL: `column < value`
	LessEq      = "lte"         // 小于等于 (`<=`)，SQL: `column <= value`
	StartsWith  = "startswith"  // 以某值开头，SQL: `LIKE 'xxx%'`
	IStartsWith = "istartswith" // 不区分大小写的以某值开头（PostgreSQL `ILIKE 'xxx%'`）
	EndsWith    = "endswith"    // 以某值结尾，SQL: `LIKE '%xxx'`
	IEndsWith   = "iendswith"   // 不区分大小写的以某值结尾（PostgreSQL `ILIKE '%xxx'`）
	In          = "in"          // IN 查询，SQL: `IN (val1, val2, val3, ...)`
	IsNull      = "isnull"      // 是否为空，SQL: `IS NULL`
	Order       = "order"       // 排序，SQL: `ORDER BY column ASC/DESC`
	LeftJoin    = "left"        // 左连接，SQL: `LEFT JOIN table ON condition`
	InnerJoins  = "inner"       // 左连接，SQL: `LEFT JOIN table ON condition`
	// 排序方式
	OrderAsc  = "asc"  // 升序排序，SQL: `ORDER BY column ASC`
	OrderDesc = "desc" // 降序排序，SQL: `ORDER BY column DESC`
)

// ResolveSearchQuery 解析
/**
 * 	exact / iexact 等于
 * 	contains / icontains 包含
 *	gt / gte 大于 / 大于等于
 *	lt / lte 小于 / 小于等于
 *	startswith / istartswith 以…起始
 *	endswith / iendswith 以…结束
 *	in
 *	isnull
 *  order 排序		e.g. order[key]=desc     order[key]=asc
 */
func ResolveSearchQuery(driver string, q interface{}, condition Condition) {
	qType := reflect.TypeOf(q)
	qValue := reflect.ValueOf(q)
	var t *resolveSearchTag
	var tag string
	var ok bool
	for i := 0; i < qType.NumField(); i++ {
		tag, ok = "", false
		tag, ok = qType.Field(i).Tag.Lookup(FromQueryTag)
		if !ok {
			//递归解析嵌套结构体
			ResolveSearchQuery(driver, qValue.Field(i).Interface(), condition)
			continue
		}
		// 跳过无效 tag
		if tag == "-" {
			continue
		}

		// 跳过无效空字段
		if qValue.Field(i).IsZero() {
			continue
		}
		t = makeTag(tag)
		// 解析查询类型
		columnRef := formatColumn(driver, t.Table, t.Column)
		switch t.Type {
		//fix Postgres 双引号
		case LeftJoin:
			//左关联
			joinSQL := formatJoinSQL(driver, t.Join, t.On, t.Table)
			join := condition.SetJoinOn(t.Type, joinSQL)
			ResolveSearchQuery(driver, qValue.Field(i).Interface(), join)
		case InnerJoins:
			//关联
			joinSQL := formatInnerJoins(driver, t.Join, t.On, t.Table)
			join := condition.SetJoinOn(t.Type, joinSQL)
			ResolveSearchQuery(driver, qValue.Field(i).Interface(), join)
		case Exact, IExact:
			condition.SetWhere(fmt.Sprintf("%s = ?", columnRef), []interface{}{qValue.Field(i).Interface()})

		case Contains, IContains:
			if driver == Postgres {
				condition.SetWhere(fmt.Sprintf(`%s ILIKE ?`, columnRef), []interface{}{"%" + qValue.Field(i).String() + "%"})
			} else {
				condition.SetWhere(fmt.Sprintf(`%s LIKE ?`, columnRef), []interface{}{"%" + qValue.Field(i).String() + "%"})
			}
		case Greater:
			condition.SetWhere(fmt.Sprintf("%s > ?", columnRef), []interface{}{qValue.Field(i).Interface()})

		case GreaterEq:
			condition.SetWhere(fmt.Sprintf("%s >= ?", columnRef), []interface{}{qValue.Field(i).Interface()})

		case Less:
			condition.SetWhere(fmt.Sprintf("%s < ?", columnRef), []interface{}{qValue.Field(i).Interface()})

		case LessEq:
			condition.SetWhere(fmt.Sprintf("%s <= ?", columnRef), []interface{}{qValue.Field(i).Interface()})

		case StartsWith, IStartsWith:
			if driver == Postgres {
				condition.SetWhere(fmt.Sprintf(`%s ILIKE ?`, columnRef), []interface{}{qValue.Field(i).String() + "%"})
			} else {
				condition.SetWhere(fmt.Sprintf(`%s LIKE ?`, columnRef), []interface{}{qValue.Field(i).String() + "%"})
			}
		case EndsWith, IEndsWith:
			if driver == Postgres {
				condition.SetWhere(fmt.Sprintf(`%s ILIKE ?`, columnRef), []interface{}{"%" + qValue.Field(i).String()})
			} else {
				condition.SetWhere(fmt.Sprintf(`%s LIKE ?`, columnRef), []interface{}{"%" + qValue.Field(i).String()})
			}
		case In:
			condition.SetWhere(fmt.Sprintf("%s in (?)", columnRef), []interface{}{qValue.Field(i).Interface()})

		case IsNull:
			if !(qValue.Field(i).IsZero() && qValue.Field(i).IsNil()) {
				condition.SetWhere(fmt.Sprintf("%s is null", columnRef), make([]interface{}, 0))
			}

		case Order:
			orderValue := strings.ToLower(qValue.Field(i).String())
			if orderValue == "desc" || orderValue == "asc" {
				condition.SetOrder(fmt.Sprintf("%s %s", columnRef, orderValue))
			}
		}
	}
}

// 处理数据库字段格式（Postgres 使用双引号，MySQL 使用反引号）
func formatColumn(driver, table, column string) string {
	if driver == Postgres {
		return fmt.Sprintf(`"%s"."%s"`, table, column)
	}
	return fmt.Sprintf("`%s`.`%s`", table, column)
}

// 生成 JOIN 语句
func formatJoinSQL(driver, joinTable string, on []string, table string) string {
	if driver == Postgres {
		return fmt.Sprintf(
			`LEFT JOIN "%s" ON "%s"."%s" = "%s"."%s"`,
			joinTable, joinTable, on[0], table, on[1],
		)
	}
	return fmt.Sprintf(
		"LEFT JOIN `%s` ON `%s`.`%s` = `%s`.`%s`",
		joinTable, joinTable, on[0], table, on[1],
	)
}

// 生成 JOIN 语句
func formatInnerJoins(driver, joinTable string, on []string, table string) string {
	if driver == Postgres {
		return fmt.Sprintf(
			`INNER JOIN "%s" ON "%s"."%s" = "%s"."%s"`,
			joinTable, joinTable, on[0], table, on[1],
		)
	}
	return fmt.Sprintf(
		"INNER JOIN `%s` ON `%s`.`%s` = `%s`.`%s`",
		joinTable, joinTable, on[0], table, on[1],
	)
}
