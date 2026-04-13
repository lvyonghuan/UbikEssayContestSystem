package pgsql

import (
	"errors"
	"main/conf"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var postgresDB *gorm.DB

func Start(dbConf conf.DBConfig) error {
	dsn := "host=" + dbConf.Host + " user=" + dbConf.User + " password=" + dbConf.Password + " port= " + dbConf.Port + " dbname= ubik TimeZone=Asia/Shanghai"

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(dbConf.LogLevel)),
	}

	gdb, err := gorm.Open(postgres.Open(dsn), gormCfg)
	if err != nil {
		return uerr.NewError(errors.New("连接数据库失败: " + err.Error()))
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return uerr.NewError(errors.New("获取数据库连接失败: " + err.Error()))
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	postgresDB = gdb
	return nil
}
