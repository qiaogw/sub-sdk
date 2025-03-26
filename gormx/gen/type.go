package gen

import (
	"github.com/pkg/errors"
	"github.com/qiaogw/sub-sdk/errx"
	"github.com/qiaogw/sub-sdk/gormx/configx"
	"gorm.io/gorm"
)

type DbService interface {
	Init(tx *gorm.DB)
	GetDB() (data []*Database, err error)
	GetTables(db string) ([]*Table, error)
	GetColumn(db, table string) (*ColumnData, error)
}

type AutoCodeService struct {
	DB        DbService
	mode      string //模式(rpc、api)
	overwrite bool   //是否覆盖
	Database  *Database
	oneMode   bool
}

func NewAutoCodeServiceByDB(tx *gorm.DB) (*AutoCodeService, error) {
	acd := new(AutoCodeService)
	switch tx.Name() {
	case string(configx.MySQL):
		acd.DB = new(Mysql)
		acd.DB.Init(tx)
	case string(configx.Postgres):
		acd.DB = new(Postgres)
		acd.DB.Init(tx)
	case string(configx.Sqlite):
		acd.DB = new(Sqlite)
		acd.DB.Init(tx)
	default:
		acd.DB = new(Mysql)
		acd.DB.Init(tx)
	}
	return acd, nil
}

func NewAutoCodeService(db *Database, one ...bool) (*AutoCodeService, error) {
	acd := AutoCodeService{}
	switch db.Driver {
	case string(configx.MySQL):
		acd.DB = new(Mysql)
	case string(configx.Sqlite):
		acd.DB = new(Postgres)
	default:
		acd.DB = new(Mysql)
	}
	dbConfig := configx.DbConf{
		Driver:      db.Driver,
		Host:        db.Host,
		Port:        db.Port,
		Config:      db.Config,
		Username:    db.Username,
		Password:    db.Password,
		Dbname:      db.Dbname,
		TablePrefix: db.TablePrefix,
	}

	tx, err := configx.GetConnect(dbConfig)
	if err != nil {
		return nil, errors.Wrapf(errx.NewErrCode(errx.DbError),
			"该数据源连接失败:%v", err)
	}
	acd.mode = db.Mode
	acd.overwrite = true //覆盖
	acd.DB.Init(tx)
	acd.Database = db
	if len(one) > 0 && one[0] {
		acd.oneMode = one[0]
	}
	return &acd, nil
}
