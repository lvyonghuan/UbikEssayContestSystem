package model

type GlobalConfig struct {
	ID               int `gorm:"primaryKey;autoIncrement" json:"id"`
	IsInit           bool
	SiteName         string
	EmailAddress     string
	EmailAppPassword string
	EmailSmtpServer  string
	EmailSmtpPort    int
}

type Permission struct {
	PermissionID int    `gorm:"primaryKey;autoIncrement" json:"permissionID"`
	Name         string `json:"name"`     // 权限名称，如 "read_user", "write_problem"
	Resource     string `json:"resource"` // 资源标识符,如 "user", "problem", "submission"
	Action       string `json:"action"`   // 操作类型，如 "read", "write", "delete"

	//保留字
	Meta map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"meta"` // 其他元信息
}

type Role struct {
	RoleID      int                    `gorm:"primaryKey;autoIncrement" json:"roleID"`
	RoleName    string                 `gorm:"primaryKey;autoIncrement" json:"roleName"`
	Description string                 `json:"description"`
	IsDefault   bool                   `json:"isDefault"` // 是否为默认角色
	IsSuper     bool                   `json:"isSuper"`   // 是否为超级管理员角色
	Meta        map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"meta"`
}

type RolePermission struct {
	RoleID       int  `json:"roleID"`
	PermissionID uint `json:"permissionID"`
}
