package stringx

import "fmt"

// UniqueKeys 对给定的字符串切片进行去重，返回唯一的字符串键列表
func UniqueKeys(keys []string) []string {
	keySet := make(map[string]struct{})
	// 将每个键存入 map 以实现去重
	for _, key := range keys {
		keySet[key] = struct{}{}
	}

	uniKeys := make([]string, 0, len(keySet))
	// 从 map 中提取唯一键
	for key := range keySet {
		uniKeys = append(uniKeys, key)
	}

	return uniKeys
}

func GetCacheKeys(cacheKeyExpire, cacheKeyExpirePrefix string, id any) []string {
	if id == nil {
		return []string{}
	}
	cacheIdKey := fmt.Sprintf("%s%v", cacheKeyExpire, id)
	cacheKeys := []string{
		cacheIdKey,
	}

	cacheKeys = append(cacheKeys, CustomCacheKeys(cacheKeyExpirePrefix, id)...)
	return cacheKeys
}

func CustomCacheKeys(cacheKeyFormat string, id any) []string {
	if id == nil {
		return []string{}
	}
	return []string{
		fmt.Sprintf("%s%v", cacheKeyFormat, id),
	}
}

// GetCacheMultiKeys 根据id生成所有缓存键
func GetCacheMultiKeys(id any, cacheKeyExpires ...string) []string {
	if id == nil {
		return []string{}
	}
	var cacheKeys []string
	for _, v := range cacheKeyExpires {
		cacheKeys = append(cacheKeys, CustomCacheKeys(v, id)...)
	}
	return cacheKeys
}

// GetCacheKeysByIdList 根据id列表生成所有缓存键，并进行去重
func GetCacheKeysByIdList(data []interface{}, cacheKeyExpires ...string) []string {
	if len(data) == 0 {
		return []string{}
	}
	var keys []string
	// 遍历每个数据，收集缓存键
	for _, v := range data {
		cacheKeys := GetCacheMultiKeys(v, cacheKeyExpires...)
		keys = append(keys, cacheKeys...)
	}
	// 去重后返回缓存键列表
	keys = UniqueKeys(keys)
	return keys
}
