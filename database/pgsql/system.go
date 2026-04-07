package pgsql

import (
	"main/conf"
	"main/model"

	"github.com/lvyonghuan/Ubik-Util/uerr"
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
