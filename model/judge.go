package model

import "time"

type Judge struct {
	JudgeID   int    `gorm:"primaryKey;autoIncrement" json:"judgeID"`
	JudgeName string `json:"judgeName"`
	Password  string `json:"password"`
}

type ReviewEvent struct {
	EventID    int       `gorm:"primaryKey;autoIncrement;column:event_id" json:"eventID"`
	TrackID    int       `json:"trackID"`
	EventName  string    `json:"eventName"`
	WorkStatus string    `json:"workStatus"`
	StartTime  time.Time `gorm:"type:timestamp" json:"startTime"`
	EndTime    time.Time `gorm:"type:timestamp" json:"endTime"`
	JudgeIDs   []int     `gorm:"-" json:"judgeIDs,omitempty"`
}

type ReviewEventJudge struct {
	EventID int `gorm:"primaryKey;column:event_id" json:"eventID"`
	JudgeID int `gorm:"primaryKey;column:judge_id" json:"judgeID"`
}
