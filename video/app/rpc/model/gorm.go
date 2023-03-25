package model

import (
	"Mini-Tiktok/video/app/rpc/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitGorm 初始化 Gorm 连接数据库
func InitGorm(c config.DbConfig) (*gorm.DB, error) {
	m := config.Mysql{DbConfig: c}
	mysqlConfig := mysql.Config{
		DSN: m.Dsn(),
	}

	db, err := gorm.Open(mysql.New(mysqlConfig))
	if err != nil {
		return nil, err
	} else {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(m.MaxIdleConns)
		sqlDB.SetMaxOpenConns(m.MaxOpenConns)
		return db, nil
	}
}
