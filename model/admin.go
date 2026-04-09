package model

import "time"

type Admin struct {
	AdminID    int    `gorm:"primaryKey;autoIncrement" json:"adminID"`
	AdminName  string `json:"adminName"`
	Password   string `json:"password"`
	AdminEmail string `json:"adminEmail"`
	IsActive   bool   `json:"isActive"`
}

type CreateSubAdminRequest struct {
	AdminName       string   `json:"adminName"`
	AdminEmail      string   `json:"adminEmail"`
	PermissionNames []string `json:"permissionNames"`
}

type UpdateSubAdminPermissionsRequest struct {
	PermissionNames []string `json:"permissionNames"`
}

type BatchCreateSubAdminsRequest struct {
	Emails          []string `json:"emails"`
	PermissionNames []string `json:"permissionNames"`
}

type HandoverSuperAdminRequest struct {
	NewSuperAdminID int `json:"newSuperAdminID"`
}

type SubAdminInfo struct {
	AdminID         int      `json:"adminID"`
	AdminName       string   `json:"adminName"`
	AdminEmail      string   `json:"adminEmail"`
	IsActive        bool     `json:"isActive"`
	PermissionNames []string `json:"permissionNames"`
}

type SubAdminCreateResult struct {
	AdminID      int    `json:"adminID"`
	AdminName    string `json:"adminName"`
	AdminEmail   string `json:"adminEmail"`
	TempPassword string `json:"tempPassword"`
	EmailSent    bool   `json:"emailSent"`
	EmailError   string `json:"emailError,omitempty"`
}

type BatchCreateSubAdminFailure struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

type BatchCreateSubAdminsResponse struct {
	Created []SubAdminCreateResult       `json:"created"`
	Failed  []BatchCreateSubAdminFailure `json:"failed"`
}

type ActionLog struct {
	LogID     int                    `gorm:"primaryKey;autoIncrement" json:"logID"`
	AdminID   int                    `json:"adminID"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	CreatedAt time.Time              `json:"createdAt"`
	Details   map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"details"`
}
