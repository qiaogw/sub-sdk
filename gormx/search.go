package gormx

import (
	"fmt"
	"github.com/qiaogw/sub-sdk/gormx/gen"
	"gorm.io/gorm"
	"reflect"
)

// SearchKey 根据表名和关键字生成SQL查询条件
// 该函数会根据数据库类型（如MySQL或PostgreSQL）生成不同的SQL查询条件
func SearchKey(db *gorm.DB, table, key string) string {
	var sql string

	genApp := new(gen.AutoCodeService)

	genApp.DB.Init(db)

	database := db.Config.NamingStrategy.SchemaName(table)
	field, err := genApp.DB.GetColumn(database, table)
	if err != nil {
		return sql
	}
	// 根据数据库类型生成SQL查询条件
	switch db.Name() {
	case "mysql":
		sql = fmt.Sprintf("concat(%v) like '%%%s%%'", field, key)
	case "postgres":
		sql = fmt.Sprintf(`CAST("%s" AS text) ~ '%s'`, table, key)
	}

	return sql
}

// MakeCondition 根据查询对象和数据库驱动生成GORM查询条件
// 该函数会解析查询对象并生成对应的GORM查询条件，包括JOIN、WHERE、OR和ORDER BY等
func MakeCondition(q interface{}, driver string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		condition := &GormCondition{
			GormPublic: GormPublic{},
			Join:       make([]*GormJoin, 0),
		}
		ResolveSearchQuery(driver, q, condition)
		for _, join := range condition.Join {
			if join == nil {
				continue
			}
			db = db.Joins(join.JoinOn)
			for k, v := range join.Where {
				db = db.Where(k, v...)
			}
			for k, v := range join.Or {
				db = db.Or(k, v...)
			}
			for _, o := range join.Order {
				db = db.Order(o)
			}
		}
		for k, v := range condition.Where {
			db = db.Where(k, v...)
		}
		for k, v := range condition.Or {
			db = db.Or(k, v...)
		}
		for _, o := range condition.Order {
			db = db.Order(o)
		}
		return db
	}
}

// Paginate 根据分页参数生成GORM分页查询条件
// 该函数会根据传入的页码和每页大小生成对应的分页查询条件
func Paginate(pageSize, pageIndex int64) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset := (pageIndex - 1) * pageSize
		if offset < 0 {
			offset = 0
		}
		return db.Offset(int(offset)).Limit(int(pageSize))
	}
}

// SortBy 根据排序字段和排序方向生成GORM排序查询条件
// 该函数会根据传入的排序字段和排序方向（升序或降序）生成对应的排序查询条件
func SortBy(sortBy string, descending bool) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		var orderBy string
		if descending {
			orderBy = sortBy + " DESC"
		} else {
			orderBy = sortBy
		}
		return db.Order(orderBy)
	}
}

// GetSortBy 根据传入的结构体和字段名获取对应的排序字段
// 该函数会从结构体的字段标签中解析出对应的排序字段，并返回该字段的名称
func GetSortBy(sort interface{}, fieldName string) string {
	tagJson := "json"
	rt := ""
	var t *resolveSearchTag
	// 获取结构体的类型
	qType := reflect.TypeOf(sort)
	// 遍历结构体的字段
	for i := 0; i < qType.NumField(); i++ {
		tag, ok := qType.Field(i).Tag.Lookup(FromQueryTag)
		if !ok {
			continue
		}
		switch tag {
		case "-":
			continue
		}
		t = makeTag(tag)
		if qType.Field(i).Tag.Get(tagJson) == fieldName {
			rt = t.Column
			break
		}
	}
	return rt
}
