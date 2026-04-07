package conf

import "github.com/spf13/viper"

type Config struct {
	API    APIConfig
	DB     DBConfig
	Redis  RedisConfig
	Log    LogConfig
	System SystemConfig
}

// SystemConfig 系统级配置
type SystemConfig struct {
	Email EmailConfig
	Token TokenConfig
}

type EmailConfig struct {
	EmailAddress     string
	EmailAPPPassword string
	SMTPHost         string
	SMTPPort         int
}

type TokenConfig struct {
	AccessTokenExpire  int
	RefreshTokenExpire int
}

type APIConfig struct {
	IsDebug bool

	// 端口号设置
	SubmissionsPort string //默认应该是80
	JudgePort       string
	AdminPort       string
	GlobalInfoPort  string //提供一些全局信息
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	LogLevel int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type LogConfig struct {
	Level       int
	WriteLevel  int
	LogFilePath string
}

// ReadConfig 使用viper读取toml配置文件
func ReadConfig() (*Config, error) {
	v := viper.New()
	v.SetConfigName("conf")
	v.SetConfigType("toml")
	v.AddConfigPath("./")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
