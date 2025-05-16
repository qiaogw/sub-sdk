package gen

import (
	"database/sql"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"sort"
)

type Sqlite struct {
	DB *gorm.DB
}

func (s *Sqlite) Init(tx *gorm.DB) {
	s.DB = tx
}

// SQLite 无需实现 GetDB（没有多个 schema）
func (s *Sqlite) GetDB() (data []*Database, err error) {
	//data = []*Database{{Database: "main"}}
	return
}

func (s *Sqlite) GetTables(_ string) ([]*Table, error) {
	var tables []struct {
		Name string
	}
	err := s.DB.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';").Scan(&tables).Error
	if err != nil {
		return nil, err
	}
	var result []*Table
	for _, t := range tables {
		result = append(result, &Table{
			Table: t.Name,
		})
	}
	return result, nil
}

// SQLite: 使用 PRAGMA table_info
func (s *Sqlite) GetColumn(db, table string) (*ColumnData, error) {
	var cols []struct {
		Cid       int            `gorm:"column:cid"`
		Name      string         `gorm:"column:name"`
		Type      string         `gorm:"column:type"`
		NotNull   int            `gorm:"column:notnull"`
		DfltValue sql.NullString `gorm:"column:dflt_value"`
		PK        int            `gorm:"column:pk"`
	}
	err := s.DB.Raw(fmt.Sprintf("PRAGMA table_info(`%s`);", table)).Scan(&cols).Error
	if err != nil {
		logx.Errorf("❌ getcolumn err: %v", err)
		return nil, err
	}

	var list []*Column
	for _, item := range cols {
		dbc := &DbColumn{
			Name:            item.Name,
			DataType:        item.Type,
			ColumnDefault:   item.DfltValue.String,
			IsNullAble:      map[bool]string{true: "YES", false: "NO"}[item.NotNull == 0],
			OrdinalPosition: item.Cid,
		}
		index, err := s.FindIndex(db, table, item.Name)
		if err != nil {
			continue
		}
		if len(index) > 0 {
			for _, i := range index {
				list = append(list, &Column{
					DbColumn: dbc,
					Index:    i,
				})
			}
		} else {
			list = append(list, &Column{
				DbColumn: dbc,
			})
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].OrdinalPosition < list[j].OrdinalPosition
	})

	return &ColumnData{
		Db:      db,
		Table:   table,
		Columns: list,
	}, nil
}

// 使用 PRAGMA index_list + index_info
func (s *Sqlite) FindIndex(db, table, column string) ([]*DbIndex, error) {
	var indexes []struct {
		Name   string `gorm:"column:name"`
		Unique int    `gorm:"column:unique"`
	}
	err := s.DB.Raw(fmt.Sprintf("PRAGMA index_list(`%s`);", table)).Scan(&indexes).Error
	if err != nil {
		return nil, err
	}

	var results []*DbIndex
	for _, idx := range indexes {
		var cols []struct {
			Seqno int    `gorm:"column:seqno"`
			Cid   int    `gorm:"column:cid"`
			Name  string `gorm:"column:name"`
		}
		_ = s.DB.Raw(fmt.Sprintf("PRAGMA index_info(`%s`);", idx.Name)).Scan(&cols).Error
		for _, col := range cols {
			if col.Name == column {
				results = append(results, &DbIndex{
					IndexName:  idx.Name,
					NonUnique:  ifThenInt(idx.Unique == 0, 1, 0),
					SeqInIndex: col.Seqno,
				})
			}
		}
	}
	return results, nil
}
func ifThenInt(cond bool, a, b int) int {
	if cond {
		return a
	}
	return b
}
