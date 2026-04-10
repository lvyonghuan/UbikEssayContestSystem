package model

type Author struct {
	AuthorID    int            `gorm:"primaryKey;autoIncrement" json:"authorID"`
	AuthorName  string         `json:"authorName"`
	PenName     string         `json:"penName"`
	Password    string         `json:"password"`
	AuthorEmail string         `json:"authorEmail"`
	AuthorInfos map[string]any `gorm:"type:jsonb;serializer:json" json:"authorInfos"`
}

type Work struct {
	WorkID     int            `gorm:"primaryKey;autoIncrement" json:"workID"`
	WorkTitle  string         `json:"workTitle"`
	TrackID    int            `json:"trackID"`
	AuthorID   int            `json:"authorID"`
	WorkStatus string         `gorm:"column:work_status" json:"workStatus"`
	AuthorName string         `gorm:"column:author_name;->" json:"authorName,omitempty"`
	TrackName  string         `gorm:"column:track_name;->" json:"trackName,omitempty"`
	WorkInfos  map[string]any `gorm:"type:jsonb;serializer:json" json:"workInfos"`
}
