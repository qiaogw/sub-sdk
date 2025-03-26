package configx

import (
	"errors"
	"github.com/glebarez/sqlite"
	"github.com/qiaogw/sub-sdk/gormx/plugins"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"path/filepath"
	"time"
)

// Sqlite3 定义了 Sqlite3 数据库的配置信息
type Sqlite3 struct {
	Driver        string
	Host          string // 文件路径
	Username      string // 数据库用户名
	Password      string // 数据库密码
	Dbname        string // 数据库名称
	TimeZone      string `json:",default=Asia/Shanghai"`                    // 数据库时区，默认 Asia/Shanghai
	SslMode       string `json:",default=disable,options=disable|enable"`   // SSL 模式，支持 disable 或 enable，默认 disable
	MaxIdleConns  int    `json:",default=10"`                               // 空闲连接数的最大值
	MaxOpenConns  int    `json:",default=10"`                               // 打开数据库连接的最大数
	LogMode       string `json:",default=dev,options=dev|test|prod|silent"` // 日志模式，取值范围为 dev、test、prod、silent，默认 dev
	LogColorful   bool   `json:",default=false"`                            // 是否启用日志彩色输出，默认 false
	SlowThreshold int64  `json:",default=1000"`                             // 慢 SQL 阈值（毫秒）
	Schema        string `json:",default=public"`
}

// Dsn 根据 Sqlite3 配置生成 PostgreSQL 的 DSN（数据源名称）
func (m *Sqlite3) Dsn() string {
	if len(m.Schema) < 1 {
		m.Schema = "public"
	}
	return filepath.Join(m.Host, m.Dbname+".db")
}

// GetGormLogMode 根据配置返回 Gorm 的日志级别
func (m *Sqlite3) GetGormLogMode() logger.LogLevel {
	return OverwriteGormLogMode(m.LogMode)
}

// GetSlowThreshold 返回慢 SQL 阈值，并转换为 time.Duration 类型（单位为毫秒）
func (m *Sqlite3) GetSlowThreshold() time.Duration {
	return time.Duration(m.SlowThreshold) * time.Millisecond
}

// GetColorful 返回是否启用日志彩色输出的配置
func (m *Sqlite3) GetColorful() bool {
	return m.LogColorful
}

// Connect 根据 Sqlite3 配置连接 PostgreSQL 数据库，返回 *gorm.DB 对象和可能出现的错误
func (m *Sqlite3) Connect() (*gorm.DB, error) {
	// 如果数据库名称为空，则返回错误
	if m.Dbname == "" {
		return nil, errors.New("database name is empty")
	}
	newLogger := NewDefaultGormLogger(m)
	db, err := gorm.Open(sqlite.Open(m.Dsn()), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			//TablePrefix:   dbCfg.Prefix,
			SingularTable: true,
		},
		Logger: newLogger,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// ConnectWithConfig 根据 Sqlite3 配置以及自定义的 gorm.Config 连接 PostgreSQL 数据库
// 并初始化插件，返回 *gorm.DB 对象和可能出现的错误
func (m *Sqlite3) ConnectWithConfig(cfg *gorm.Config) (*gorm.DB, error) {
	// 如果数据库名称为空，则返回错误
	if m.Dbname == "" {
		return nil, errors.New("database name is empty")
	}
	cfg.Logger = NewDefaultZeroLogger(m)
	cfg.NamingStrategy = schema.NamingStrategy{
		SingularTable: true,
	}
	db, err := gorm.Open(sqlite.Open(m.Dsn()), cfg)
	if err != nil {
		return nil, err
	}
	if err := plugins.InitPlugins(db); err != nil {
		return nil, err
	}
	logx.Infof("SQLite 数据库连接成功：%s", m.Dsn())
	return db, nil
}
