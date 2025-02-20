package config

import (
	"errors"
	"fmt"
	"github.com/zhanghaidi/zero-common/define"
	"gorm.io/gorm/schema"
	"io"
	"os"
	"path/filepath"
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
	LogMode       string `json:",default=error,env=DATABASE_LOG_MODE"`            // 日志级别
	EnableLogFile bool   `json:",default=false,env=DATABASE_ENABLE_LOG_FILE"`     // 是否启用日志文件
	LogFilePath   string `json:",default=logs/db.log,env=DATABASE_LOG_FILE_PATH"` // 日志文件路径
}

// InitDatabase 初始化数据库连接
func (c DatabaseConf) InitDatabase() (*gorm.DB, error) {
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
			newDBLogger(c.EnableLogFile, c.LogFilePath),
			logger.Config{
				SlowThreshold:             500 * time.Millisecond,     // 慢 SQL 阈值
				LogLevel:                  getGormLogLevel(c.LogMode), // 日志级别
				IgnoreRecordNotFoundError: true,                       // 忽略 ErrRecordNotFound 错误
				Colorful:                  !c.EnableLogFile,           // 是否禁用彩色打印
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

// getGormLogLevel 根据配置获取 Gorm 日志级别
func getGormLogLevel(logMode string) logger.LogLevel {
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

type GormLogger struct {
	writer io.Writer
}

// Printf 实现 GORM 的 logger.Writer 接口
func (l GormLogger) Printf(format string, v ...interface{}) {
	fmt.Fprintf(l.writer, format, v...)
}

func newDBLogger(enableFile bool, filePath string) logger.Writer {
	if enableFile {
		dir := filepath.Dir(filePath) // 取日志文件所在目录
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Println("创建日志目录失败:", err)
			return GormLogger{writer: os.Stdout}
		}

		// 尝试创建日志文件
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Println("创建日志文件失败:", err)
			return GormLogger{writer: os.Stdout}
		}

		// 这里可以使用 log.New() 直接创建日志记录器
		return GormLogger{writer: io.MultiWriter(file, os.Stdout)}
	}

	// 默认输出到控制台
	return GormLogger{writer: os.Stdout}
}
