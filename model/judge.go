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
	EventID    int        `gorm:"primaryKey;column:event_id" json:"eventID"`
	JudgeID    int        `gorm:"primaryKey;column:judge_id" json:"judgeID"`
	DeadlineAt *time.Time `gorm:"column:deadline_at" json:"deadlineAt,omitempty"`
}

func (ReviewEventJudge) TableName() string {
	return "review_event_judges"
}

type Review struct {
	ReviewID       int            `gorm:"primaryKey;autoIncrement;column:review_id" json:"reviewID"`
	WorkID         int            `gorm:"column:work_id" json:"workID"`
	ReviewEventID  int            `gorm:"column:review_event_id" json:"reviewEventID"`
	JudgeID        int            `gorm:"column:judge_id" json:"judgeID"`
	WorkReviews    map[string]any `gorm:"column:work_reviews;type:jsonb;serializer:json" json:"workReviews"`
}

func (Review) TableName() string {
	return "reviews"
}

type ReviewResult struct {
	ResultID       int            `gorm:"primaryKey;autoIncrement;column:result_id" json:"resultID"`
	WorkID         int            `gorm:"column:work_id" json:"workID"`
	ReviewEventID  int            `gorm:"column:review_event_id" json:"reviewEventID"`
	Reviews        map[string]any `gorm:"column:reviews;type:jsonb;serializer:json" json:"reviews"`
}

func (ReviewResult) TableName() string {
	return "review_results"
}
