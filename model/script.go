package model

import "time"

// ScriptDefinition is the base script metadata managed by admin.
type ScriptDefinition struct {
	ScriptID    int            `gorm:"primaryKey;autoIncrement" json:"scriptID"`
	ScriptKey   string         `gorm:"uniqueIndex;size:255;not null" json:"scriptKey"`
	ScriptName  string         `gorm:"size:255;not null" json:"scriptName"`
	Interpreter string         `gorm:"size:64;not null" json:"interpreter"`
	Description string         `gorm:"type:text" json:"description"`
	IsEnabled   bool           `gorm:"default:true" json:"isEnabled"`
	Meta        map[string]any `gorm:"type:jsonb;serializer:json" json:"meta"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// ScriptVersion stores immutable versioned files under scripts/{script_key}/v{n}/.
type ScriptVersion struct {
	VersionID    int       `gorm:"primaryKey;autoIncrement" json:"versionID"`
	ScriptID     int       `gorm:"index;not null" json:"scriptID"`
	VersionNum   int       `gorm:"not null" json:"versionNum"`
	FileName     string    `gorm:"size:255;not null" json:"fileName"`
	RelativePath string    `gorm:"type:text;not null" json:"relativePath"`
	Checksum     string    `gorm:"size:128" json:"checksum"`
	IsActive     bool      `gorm:"default:false" json:"isActive"`
	CreatedBy    int       `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ScriptFlow defines a reusable chain of script steps.
type ScriptFlow struct {
	FlowID      int            `gorm:"primaryKey;autoIncrement" json:"flowID"`
	FlowKey     string         `gorm:"uniqueIndex;size:255;not null" json:"flowKey"`
	FlowName    string         `gorm:"size:255;not null" json:"flowName"`
	Description string         `gorm:"type:text" json:"description"`
	IsEnabled   bool           `gorm:"default:true" json:"isEnabled"`
	Meta        map[string]any `gorm:"type:jsonb;serializer:json" json:"meta"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// FlowStep defines one execution unit in a flow.
type FlowStep struct {
	StepID          int            `gorm:"primaryKey;autoIncrement" json:"stepID"`
	FlowID          int            `gorm:"index;not null" json:"flowID"`
	StepOrder       int            `gorm:"not null" json:"stepOrder"`
	StepName        string         `gorm:"size:255;not null" json:"stepName"`
	ScriptID        int            `gorm:"not null" json:"scriptID"`
	ScriptVersionID int            `json:"scriptVersionID"`
	TimeoutMs       int            `gorm:"default:5000" json:"timeoutMs"`
	FailureStrategy string         `gorm:"size:32;default:'fail_close'" json:"failureStrategy"`
	InputTemplate   map[string]any `gorm:"type:jsonb;serializer:json" json:"inputTemplate"`
	IsEnabled       bool           `gorm:"default:true" json:"isEnabled"`
}

func (FlowStep) TableName() string {
	return "script_flow_steps"
}

// FlowMount binds a flow to scope/event/target.
type FlowMount struct {
	MountID    int       `gorm:"primaryKey;autoIncrement" json:"mountID"`
	FlowID     int       `gorm:"index;not null" json:"flowID"`
	Scope      string    `gorm:"size:64;not null" json:"scope"`
	EventKey   string    `gorm:"size:128;not null" json:"eventKey"`
	TargetType string    `gorm:"size:64;not null" json:"targetType"`
	TargetID   int       `gorm:"not null" json:"targetID"`
	IsEnabled  bool      `gorm:"default:true" json:"isEnabled"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (FlowMount) TableName() string {
	return "script_flow_mounts"
}
