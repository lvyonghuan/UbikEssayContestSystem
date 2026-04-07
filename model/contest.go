package model

import "gorm.io/datatypes"

// Contest 比赛模型
type Contest struct {
	ContestID           int            `gorm:"primaryKey;autoIncrement" json:"contestID"`
	ContestName         string         `json:"contestName"`         //比赛名称
	ContestStartDate    datatypes.Date `json:"contestStartDate"`    //比赛开始日期，精确到分
	ContestEndDate      datatypes.Date `json:"contestEndDate"`      //比赛结束日期，精确到分
	ContestIntroduction string         `json:"contestIntroduction"` //比赛简介
}

type Track struct {
	TrackID          int            `gorm:"primaryKey;autoIncrement" json:"trackID"`
	TrackName        string         `json:"trackName"` //赛道名称
	ContestID        int            `json:"contestID"`
	TrackDescription string         `json:"trackDescription"`                                //赛道描述
	TrackSettings    map[string]any `gorm:"type:jsonb;serializer:json" json:"trackSettings"` //赛道设置，存储为JSON格式
}
