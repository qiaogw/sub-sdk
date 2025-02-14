package utils

import (
	"reflect"
	"strings"
)

type resolveSearchTag struct {
	Type   string
	Column string
	Table  string
	On     []string
	Join   string
}

func GetSortBy(sort interface{}, fieldName string) string {
	FromQueryTag := "search" // 在标签中搜索的标识符
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

// makeTag 解析search的tag标签
func makeTag(tag string) *resolveSearchTag {
	r := &resolveSearchTag{}
	tags := strings.Split(tag, ";")
	var ts []string
	for _, t := range tags {
		ts = strings.Split(t, ":")
		if len(ts) == 0 {
			continue
		}
		switch ts[0] {
		case "type":
			if len(ts) > 1 {
				r.Type = ts[1]
			}
		case "column":
			if len(ts) > 1 {
				r.Column = ts[1]
			}
		case "table":
			if len(ts) > 1 {
				r.Table = ts[1]
			}
		case "on":
			if len(ts) > 1 {
				r.On = ts[1:]
			}
		case "join":
			if len(ts) > 1 {
				r.Join = ts[1]
			}
		}
	}
	return r
}
