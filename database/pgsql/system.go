package pgsql

import (
	"main/conf"
	"main/model"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

func getGlobalConfig() (model.GlobalConfig, error) {
	var globalConfig model.GlobalConfig

	result := postgresDB.Table("global_config").First(&globalConfig)
	if result.Error != nil {
		return model.GlobalConfig{}, uerr.NewError(result.Error)
	}

	return globalConfig, nil
}

// CheckIfSystemInit 检查系统是否已经初始化
func CheckIfSystemInit() (bool, error) {
	globalConfig, err := getGlobalConfig()
	if err != nil {
		return false, err
	}

	return globalConfig.IsInit, nil
}

// ChangeSystemInitStatus 修改系统初始化状态
func ChangeSystemInitStatus(isInit bool) error {
	globalConfig, err := getGlobalConfig()
	if err != nil {
		return err
	}

	globalConfig.IsInit = isInit

	err = postgresDB.Table("global_config").Save(globalConfig).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func WriteSystemEmailConfig(emailConf conf.EmailConfig) error {
	globalConfig, err := getGlobalConfig()
	if err != nil {
		return err
	}

	globalConfig.EmailAddress = emailConf.EmailAddress
	globalConfig.EmailAppPassword = emailConf.EmailAPPPassword
	globalConfig.EmailSmtpServer = emailConf.SMTPHost
	globalConfig.EmailSmtpPort = emailConf.SMTPPort

	result := postgresDB.Table("global_config").Where("id = ?", globalConfig.ID).Updates(globalConfig)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

// GetContestList 获取比赛列表
func GetContestList() ([]model.Contest, error) {
	var contests []model.Contest
	result := postgresDB.Find(&contests)
	if result.Error != nil {
		return nil, uerr.NewError(result.Error)
	}

	return contests, nil
}

// GetTrackList 获取赛道列表
func GetTrackList(contestID int) ([]model.Track, error) {
	var tracks []model.Track
	result := postgresDB.Model(&model.Track{}).Where("contest_id = ?", contestID).Find(&tracks)
	if result.Error != nil {
		return nil, uerr.NewError(result.Error)
	}
	return tracks, nil
}

func GetContestByID(contestID int) (model.Contest, error) {
	var contest model.Contest
	result := postgresDB.Where("contest_id = ?", contestID).First(&contest)
	if result.Error != nil {
		return model.Contest{}, uerr.NewError(result.Error)
	}

	return contest, nil
}

func GetTrackByID(trackID int) (model.Track, error) {
	var track model.Track
	result := postgresDB.Where("track_id = ?", trackID).First(&track)
	if result.Error != nil {
		return model.Track{}, uerr.NewError(result.Error)
	}

	return track, nil
}
func GetTracksByContestID(contestID int) ([]model.Track, error) {
	var tracks []model.Track
	result := postgresDB.Where("contest_id = ?", contestID).Find(&tracks)
	if result.Error != nil {
		return nil, uerr.NewError(result.Error)
	}

	return tracks, nil
}

func MarkTrackContestEndRunning(trackID int, triggerSource string) error {
	now := time.Now().UTC()
	source := normalizeContestEndTriggerSource(triggerSource)

	result := postgresDB.Model(&model.Track{}).Where("track_id = ?", trackID).Updates(map[string]any{
		"contest_end_status":           "running",
		"contest_end_attempt_count":    gorm.Expr("COALESCE(contest_end_attempt_count, 0) + 1"),
		"contest_end_last_error":       "",
		"contest_end_last_started_at":  now,
		"contest_end_last_finished_at": nil,
		"contest_end_trigger_source":   source,
		"contest_end_updated_at":       now,
	})
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}
	if result.RowsAffected == 0 {
		return uerr.NewError(gorm.ErrRecordNotFound)
	}

	return nil
}

func MarkTrackContestEndSuccess(trackID int, triggerSource string) error {
	now := time.Now().UTC()
	source := normalizeContestEndTriggerSource(triggerSource)

	result := postgresDB.Model(&model.Track{}).Where("track_id = ?", trackID).Updates(map[string]any{
		"contest_end_status":           "success",
		"contest_end_last_error":       "",
		"contest_end_last_finished_at": now,
		"contest_end_trigger_source":   source,
		"contest_end_updated_at":       now,
	})
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}
	if result.RowsAffected == 0 {
		return uerr.NewError(gorm.ErrRecordNotFound)
	}

	return nil
}

func MarkTrackContestEndFailed(trackID int, triggerSource string, lastError string) error {
	now := time.Now().UTC()
	source := normalizeContestEndTriggerSource(triggerSource)

	result := postgresDB.Model(&model.Track{}).Where("track_id = ?", trackID).Updates(map[string]any{
		"contest_end_status":           "failed",
		"contest_end_last_error":       strings.TrimSpace(lastError),
		"contest_end_last_finished_at": now,
		"contest_end_trigger_source":   source,
		"contest_end_updated_at":       now,
	})
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}
	if result.RowsAffected == 0 {
		return uerr.NewError(gorm.ErrRecordNotFound)
	}

	return nil
}

func MarkTrackContestEndReplayRequested(trackID int, triggerSource string) error {
	now := time.Now().UTC()
	source := normalizeContestEndTriggerSource(triggerSource)

	result := postgresDB.Model(&model.Track{}).Where("track_id = ?", trackID).Updates(map[string]any{
		"contest_end_status":         "replay_requested",
		"contest_end_last_error":     "",
		"contest_end_trigger_source": source,
		"contest_end_updated_at":     now,
	})
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}
	if result.RowsAffected == 0 {
		return uerr.NewError(gorm.ErrRecordNotFound)
	}

	return nil
}

func ResetContestEndExecutionByContest(contestID int) error {
	now := time.Now().UTC()

	result := postgresDB.Model(&model.Track{}).Where("contest_id = ?", contestID).Updates(map[string]any{
		"contest_end_status":           "pending",
		"contest_end_attempt_count":    0,
		"contest_end_last_error":       "",
		"contest_end_last_started_at":  nil,
		"contest_end_last_finished_at": nil,
		"contest_end_trigger_source":   "system",
		"contest_end_updated_at":       now,
	})
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func normalizeContestEndTriggerSource(triggerSource string) string {
	value := strings.TrimSpace(triggerSource)
	if value == "" {
		return "system"
	}

	return value
}
