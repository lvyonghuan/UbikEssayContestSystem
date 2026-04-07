package pgsql

import (
	"main/model"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

// 管理员个人账户操作 ----------------------------------------------

func FindAdminByUsername(username string) (model.Admin, error) {
	var admin model.Admin
	err := postgresDB.Where("admin_name = ?", username).First(&admin).Error
	if err != nil {
		return model.Admin{}, uerr.NewError(err)
	}

	return admin, nil
}

func ChangeAdminPassword(adminID int, newPassword string) error {
	result := postgresDB.Model(&model.Admin{}).Where("admin_id = ?", adminID).Update("password", newPassword)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

// 管理行为 ----------------------------------------------

func CreateContest(contest *model.Contest) error {
	err := postgresDB.Create(contest).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func UpdateContest(contestID int, updatedContest *model.Contest) error {
	result := postgresDB.Model(&model.Contest{}).Where("contest_id = ?", contestID).Updates(updatedContest)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func DeleteContest(contestID int) (model.Contest, error) {
	var contest model.Contest
	result := postgresDB.Where("contest_id = ?", contestID).First(&contest)
	if result.Error != nil {
		return model.Contest{}, uerr.NewError(result.Error)
	}

	result = postgresDB.Where("contest_id = ?", contestID).Delete(&model.Contest{})
	if result.Error != nil {
		return model.Contest{}, uerr.NewError(result.Error)
	}

	return contest, nil
}

func CreateTrack(track *model.Track) error {
	err := postgresDB.Create(track).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func UpdateTrack(trackID int, updatedTrack *model.Track) error {
	result := postgresDB.Model(&model.Track{}).Where("track_id = ?", trackID).Updates(updatedTrack)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func DeleteTrack(trackID int) (model.Track, error) {
	var track model.Track
	result := postgresDB.Where("track_id = ?", trackID).First(&track)
	if result.Error != nil {
		return model.Track{}, uerr.NewError(result.Error)
	}

	result = postgresDB.Where("track_id = ?", trackID).Delete(&model.Track{})
	if result.Error != nil {
		return model.Track{}, uerr.NewError(result.Error)
	}

	return track, nil
}

// 管理日志 ----------------------------------------------

// CreateActionLog 创建管理行为日志
func CreateActionLog(actionLog model.ActionLog) error {
	err := postgresDB.Create(&actionLog).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func GetWorkByID(workID int) (model.Work, error) {
	return getSubmissionByID(workID)
}

func DeleteWorkByID(workID int) error {
	work, err := getSubmissionByID(workID)
	if err != nil {
		return err
	}

	return DeleteWork(&work)
}
