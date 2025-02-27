package gormx

import "gorm.io/gorm"

// GetMaxSort 获取最大排序
func GetMaxSort(db *gorm.DB, tableName string) (any, error) {
	var maxSort any
	err := db.Table(tableName).Select("MAX(sort)").Scan(&maxSort).Error
	return maxSort, err
}

// CalculateSort 计算新 `sort` 值，确保插入到合适位置
func CalculateSort(tx *gorm.DB, tableName string, sort float64) (float64, error) {
	if sort <= 0 {
		// 1. 获取最大 `sort`，用于默认新增排序
		var maxSort float64
		err := tx.Table(tableName).
			Select("COALESCE(MAX(sort), 0)").
			Scan(&maxSort).Error
		if err != nil {
			return 0, err
		}
		return maxSort + 1, nil
	}

	// 2. 检查 `sort` 是否已存在
	var existingSort float64
	err := tx.Table(tableName).
		Select("sort").
		Where("sort = ?", sort).
		Limit(1).
		Scan(&existingSort).Error
	if err != nil || existingSort == 0 {
		// 如果 `sort` 没有重复，直接使用
		return sort, nil
	}

	// 3. 获取前一个 `sort`（比当前小的最大值）
	var prevSort, nextSort float64
	tx.Table(tableName).
		Select("MAX(sort)").
		Where("sort < ?", sort).
		Scan(&prevSort)

	// 4. 获取后一个 `sort`（比当前大的最小值）
	tx.Table(tableName).
		Select("MIN(sort)").
		Where("sort > ?", sort).
		Scan(&nextSort)

	// 5. 计算新 `sort`
	var newSort float64
	// 计算新 `sort` 值
	if prevSort > 0 && nextSort > 0 { // **情况 1**: 既有大的，也有小的
		newSort = (prevSort + sort) / 2
	} else if prevSort > 0 { // **情况 2**: 只有小的
		newSort = newSort + 1
	} else if nextSort > 0 { // **情况 3**: 只有大的
		newSort = newSort / 2
	} else { // **情况 4**: 只有一条记录
		newSort = newSort / 2
	}

	return newSort, nil
}
