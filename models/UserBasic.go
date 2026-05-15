package models

import (
	"time"

	"gorm.io/gorm"
)

type UserBasic struct {
	gorm.Model
	Name          string
	Password      string
	Phone         string
	Email         string
	Identity      string
	ClintIp       string
	ClinetPort    string
	CreatedAt     time.Time
	LoginTime     time.Time
	HeartbeatTime time.Time
	LoginOutTime  time.Time
	Islogout      bool
	DeviceInfo    string
}

func (table *UserBasic) TableName() string {
	return "user_basic"
}
