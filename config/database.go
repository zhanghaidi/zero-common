package config

import (
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zhanghaidi/zero-common/define"
	"gorm.io/gorm/schema"
	"io"
	"log"
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
	Type          string `json:",default=mysql,options=[mysql,postgres,sqlite3],env=DATABASE_TYPE"`
	Host          string `json:",env=DATABASE_HOST"`
	Port          int    `json:",env=DATABASE_PORT"`
	Username      string `json:",default=root,env=DATABASE_USERNAME"`
	Password      string `json:",optional,env=DATABASE_PASSWORD"`
	DBName        string `json:",default=test_db,env=DATABASE_DBNAME"`
	Config        string `json:",optional,env=DATABASE_CONFIG"`
	MaxIdleConn   int    `json:",optional,default=10,env=DATABASE_MAX_IDLE_CONN"`
	MaxOpenConn   int    `json:",optional,default=100,env=DATABASE_MAX_OPEN_CONN"`
	ConnMaxLife   int    `json:",optional,default=3600,env=DATABASE_CONN_MAX_LIFE"`
	Prefix        string `json:",default=cmf_,env=DATABASE_PREFIX"`
	DBPath        string `json:",optional,env=DATABASE_DBPATH"`
	LogMode       string `json:",default=error,env=DATABASE_LOG_MODE"`        // 日志级别
	EnableLogFile bool   `json:",default=false,env=DATABASE_ENABLE_LOG_FILE"` // 是否启用日志文件
	LogFilename   string `json:",default=db.log,env=DATABASE_LOG_FILENAME"`   // 日志文件名称
}

// InitDatabase 初始化数据库连接
func (c DatabaseConf) InitDatabase(conf logx.LogConf) (*gorm.DB, error) {
	if err := c.Check(); err != nil {
		return nil, fmt.Errorf("数据库配置错误: %v", err)
	}

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
		// 命名策略
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   c.Prefix, // 表前缀，在表名前添加前缀，如添加用户模块的表前缀 user_
			SingularTable: false,    // 是否使用单数形式的表名，如果设置为 true，那么 User 模型表将使用单数表名 user
		},
		DisableForeignKeyConstraintWhenMigrating: true, // 禁用自动创建外键约束
		// 配置sql日志
		Logger: logger.New(
			newGormLogger(c.EnableLogFile, c.LogFilename, conf), // 创建日志 Writer
			logger.Config{
				SlowThreshold:             500 * time.Millisecond, // 慢 SQL 阈值
				LogLevel:                  getLogLevel(c.LogMode), // 日志级别
				IgnoreRecordNotFoundError: true,                   // 忽略 ErrRecordNotFound 错误
				Colorful:                  !c.EnableLogFile,       // 是否禁用彩色打印
			}),
	})
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %v", err)
	}

	sqlDB.SetMaxOpenConns(c.MaxOpenConn) // 设置连接池最大连接数
	sqlDB.SetMaxIdleConns(c.MaxIdleConn) // 设置连接池最大空闲连接数
	sqlDB.SetConnMaxLifetime(time.Duration(c.ConnMaxLife) * time.Second)

	// 连接测试
	if err = sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	define.GlobalDatabase = db // 设置全局数据库配置
	return db, nil
}

func (c DatabaseConf) Check() error {
	if c.Type == "sqlite3" && c.DBPath == "" {
		return errors.New("SQLite 需要配置 DBPath")
	}
	if c.Type != "sqlite3" && (c.Host == "" || c.Port == 0) {
		return errors.New("MySQL 或 PostgreSQL 需要 Host 和 Port")
	}
	return nil
}

// GetDSN 根据数据库类型返回 DSN
func (c DatabaseConf) GetDSN() string {
	switch c.Type {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", c.Username, c.Password, c.Host, c.Port, c.DBName, c.Config)
	case "postgres":
		return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d %s", c.Host, c.Username, c.Password,
			c.DBName, c.Port, c.Config)
	case "sqlite3":
		if c.DBPath == "" {
			fmt.Println("数据库文件路径不能为空")
			return ""
		}
		if _, err := os.Stat(c.DBPath); os.IsNotExist(err) {
			f, err := os.OpenFile(c.DBPath, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				fmt.Printf("创建 SQLite 数据库文件失败: %s (%q)\n", err.Error(), c.DBPath)
				return ""
			}
			_ = f.Close()
		}
		return fmt.Sprintf("file:%s?_busy_timeout=100000&_fk=1%s", c.DBPath, c.Config)
	default:
		fmt.Println("未知的数据库类型")
		return ""
	}
}

// getLogLevel 根据配置获取 Gorm 日志级别
func getLogLevel(logMode string) logger.LogLevel {
	levels := map[string]logger.LogLevel{
		"info":   logger.Info,
		"warn":   logger.Warn,
		"error":  logger.Error,
		"silent": logger.Silent,
	}
	if level, exists := levels[logMode]; exists {
		return level
	}
	fmt.Printf("警告: 未知的日志级别 %q，默认使用 'error'\n", logMode)
	return logger.Error
}

// newGormLogger 创建 Gorm 日志 Writer
func newGormLogger(enableFile bool, filename string, conf logx.LogConf) logger.Writer {
	var writer io.Writer
	// 是否启用日志文件
	if enableFile {
		// 使用 go-zero 的日志功能创建日志文件 Writer
		filePath := conf.Path + "/" + filename
		writer, _ = logx.NewLogger(filePath, logx.DefaultRotateRule(filePath, "-", conf.KeepDays, conf.Compress), conf.Compress)
	} else {
		// 默认输出到控制台
		writer = os.Stdout
	}

	return log.New(writer, "\r\n", log.LstdFlags)
}
