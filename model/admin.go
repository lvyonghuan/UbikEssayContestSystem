package model

import "time"

type Admin struct {
	AdminID    int    `gorm:"primaryKey;autoIncrement" json:"adminID"`
	AdminName  string `json:"adminName"`
	Password   string `json:"password"`
	AdminEmail string `json:"adminEmail"`
}

type ActionLog struct {
	LogID     int                    `gorm:"primaryKey;autoIncrement" json:"logID"`
	AdminID   int                    `json:"adminID"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	CreatedAt time.Time              `json:"createdAt"`
	Details   map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"details"`
}
