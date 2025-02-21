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
