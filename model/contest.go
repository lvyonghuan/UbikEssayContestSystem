package model

import "time"

// Contest 比赛模型
type Contest struct {
	ContestID           int       `gorm:"primaryKey;autoIncrement" json:"contestID"`
	ContestName         string    `json:"contestName"`                            //比赛名称
	ContestStartDate    time.Time `gorm:"type:timestamp" json:"contestStartDate"` //比赛开始时间，TIMESTAMP
	ContestEndDate      time.Time `gorm:"type:timestamp" json:"contestEndDate"`   //比赛结束时间，TIMESTAMP
	ContestIntroduction string    `json:"contestIntroduction"`                    //比赛简介
}

type Track struct {
	TrackID                  int            `gorm:"primaryKey;autoIncrement" json:"trackID"`
	TrackName                string         `json:"trackName"` //赛道名称
	ContestID                int            `json:"contestID"`
	TrackDescription         string         `json:"trackDescription"`                                //赛道描述
	TrackSettings            map[string]any `gorm:"type:jsonb;serializer:json" json:"trackSettings"` //赛道设置，存储为JSON格式
	ContestEndStatus         string         `gorm:"column:contest_end_status" json:"-"`
	ContestEndAttemptCount   int            `gorm:"column:contest_end_attempt_count" json:"-"`
	ContestEndLastError      string         `gorm:"column:contest_end_last_error" json:"-"`
	ContestEndLastStartedAt  *time.Time     `gorm:"column:contest_end_last_started_at;type:timestamp" json:"-"`
	ContestEndLastFinishedAt *time.Time     `gorm:"column:contest_end_last_finished_at;type:timestamp" json:"-"`
	ContestEndTriggerSource  string         `gorm:"column:contest_end_trigger_source" json:"-"`
	ContestEndUpdatedAt      time.Time      `gorm:"column:contest_end_updated_at;type:timestamp" json:"-"`
}
