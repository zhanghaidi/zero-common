package config

type Database struct {
	Type                string `json:",default=mysql,options=[mysql,postgres]"` // type of database: mysql, postgres
	Host                string `json:",default=localhost"`                      // address
	Port                int    `json:",default=3306"`                           // port
	Config              string `json:",optional"`                               // extra config such as charset=utf8mb4&parseTime=True
	DBName              string `json:",default=histology"`                      // database name
	Username            string `json:",default=histology"`                      // username
	Password            string `json:",optional"`                               // password
	Prefix              string `json:",default=cmf_"`                           // table prefix
	MaxIdleConn         int    `json:",default=10"`                             // the maximum number of connections in the idle connection pool
	MaxOpenConn         int    `json:",default=100"`                            // the maximum number of open connections to the database
	EnableFileLogWriter bool   `json:",default=false"`                          // enable file log writer
	LogMode             string `json:",default=error"`                          // log level
	LogFilename         string `json:",default=sql.log"`                        // log file path
}

type LogConf struct {
	ServiceName         string `json:",optional"`
	Mode                string `json:",default=console,options=[console,file,volume]"`
	Encoding            string `json:",default=json,options=[json,plain]"`
	TimeFormat          string `json:",optional"`
	Path                string `json:",default=logs"`
	Level               string `json:",default=info,options=[debug,info,error,severe]"`
	MaxContentLength    uint32 `json:",optional"`
	Compress            bool   `json:",optional"`
	Stat                bool   `json:",default=true"`
	KeepDays            int    `json:",optional"`
	StackCooldownMillis int    `json:",default=100"`
	MaxBackups          int    `json:",default=0"`
	MaxSize             int    `json:",default=0"`
	Rotation            string `json:",default=daily,options=[daily,size]"`
}
