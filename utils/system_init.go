package utils

import (
	"fmt"
	"ginchat/models"
	"strings"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitConfig() {
	viper.SetConfigName("app")
	viper.AddConfigPath("./config")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("config app", viper.Get("app"))
	fmt.Println("config mysql:", viper.Get("mysql"))
}

func normalizeMySQLDSN(dsn string) string {
	if strings.Contains(dsn, "parseTime=") && strings.Contains(dsn, "loc=") {
		return dsn
	}

	if strings.Contains(dsn, "?") {
		return dsn + "&parseTime=true&loc=Local"
	}
	return dsn + "?parseTime=true&loc=Local"
}

func InitMySQL() {
	var err error
	dsn := normalizeMySQLDSN(viper.GetString("mysql.dsn"))
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	if err != nil {
		fmt.Println("连接失败", err)
		return
	}

	if DB.Migrator().HasTable(&models.Document{}) {
		if DB.Migrator().HasColumn(&models.Document{}, "Summary") {
			if err := DB.Exec("ALTER TABLE documents MODIFY COLUMN summary TEXT").Error; err != nil {
				fmt.Println("修改 summary 类型失败", err)
				return
			}
		}
		if DB.Migrator().HasColumn(&models.Document{}, "Content") {
			if err := DB.Exec("ALTER TABLE documents MODIFY COLUMN content LONGTEXT NOT NULL").Error; err != nil {
				fmt.Println("修改 content 类型失败", err)
				return
			}
		}
	}

	if err := DB.AutoMigrate(
		&models.UserBasic{},
		&models.Document{},
		&models.Tag{},
		&models.DocumentTag{},
		&models.SearchRecord{},
		&models.SearchEvaluation{}, // 新增这行，用于自动创建评价表
	); err != nil {
		fmt.Println("数据库自动迁移失败", err)
		return
	}

	SearchIndex = NewInvertedIndex()
	if err := SearchIndex.BuildFromDB(); err != nil {
		fmt.Println("搜索索引初始化失败", err)
		return
	}

	fmt.Println("数据库连接成功，自动迁移完成，搜索索引已构建")
}
