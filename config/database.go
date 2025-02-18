package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseConf 结构体存储数据库配置
type DatabaseConf struct {
	Host         string `json:",env=DATABASE_HOST"`
	Port         int    `json:",env=DATABASE_PORT"`
	Username     string `json:",default=root,env=DATABASE_USERNAME"`
	Password     string `json:",optional,env=DATABASE_PASSWORD"`
	DBName       string `json:",default=test_db,env=DATABASE_DBNAME"`
	SSLMode      string `json:",optional,env=DATABASE_SSL_MODE"`
	Type         string `json:",default=mysql,options=[mysql,postgres,sqlite3],env=DATABASE_TYPE"`
	MaxOpenConn  int    `json:",optional,default=100,env=DATABASE_MAX_OPEN_CONN"`
	MaxIdleConn  int    `json:",optional,default=10,env=DATABASE_MAX_IDLE_CONN"`
	ConnMaxLife  int    `json:",optional,default=3600,env=DATABASE_CONN_MAX_LIFE"` // 单位: 秒
	DBPath       string `json:",optional,env=DATABASE_DBPATH"`
	MysqlConfig  string `json:",optional,env=DATABASE_MYSQL_CONFIG"`
	PGConfig     string `json:",optional,env=DATABASE_PG_CONFIG"`
	SqliteConfig string `json:",optional,env=DATABASE_SQLITE_CONFIG"`
}

// InitDatabase 初始化数据库连接，返回 *gorm.DB
func (c DatabaseConf) InitDatabase() (*gorm.DB, error) {
	dsn := c.GetDSN()
	if dsn == "" {
		return nil, errors.New("数据库 DSN 不能为空")
	}

	var dialector gorm.Dialector
	switch c.Type {
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	case "sqlite3":
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", c.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 默认日志级别
	})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %v", err)
	}

	sqlDB.SetMaxOpenConns(c.MaxOpenConn)
	sqlDB.SetMaxIdleConns(c.MaxIdleConn)
	sqlDB.SetConnMaxLifetime(time.Duration(c.ConnMaxLife) * time.Second)

	fmt.Println("成功连接到数据库:", c.DBName)
	return db, nil
}

// GetDSN 根据数据库类型返回 DSN
func (c DatabaseConf) GetDSN() string {
	switch c.Type {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local%s",
			c.Username, c.Password, c.Host, c.Port, c.DBName, c.MysqlConfig)
	case "postgres":
		return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s%s",
			c.Username, c.Password, c.Host, c.Port, c.DBName, c.SSLMode, c.PGConfig)
	case "sqlite3":
		if c.DBPath == "" {
			fmt.Println("数据库文件路径不能为空")
			return ""
		}
		// 如果数据库文件不存在，则创建它
		if _, err := os.Stat(c.DBPath); os.IsNotExist(err) {
			f, err := os.OpenFile(c.DBPath, os.O_CREATE|os.O_RDWR, 0600)
			if err != nil {
				fmt.Printf("创建 SQLite 数据库文件失败: %q\n", c.DBPath)
				return ""
			}
			_ = f.Close()
		} else {
			_ = os.Chmod(c.DBPath, 0660)
		}
		return fmt.Sprintf("file:%s?_busy_timeout=100000&_fk=1%s", c.DBPath, c.SqliteConfig)
	default:
		fmt.Println("未知的数据库类型")
		return ""
	}
}
