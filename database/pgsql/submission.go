package pgsql

import (
	"main/model"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/datatypes"
)

func SubmissionWork(work *model.Work) error {
	err := postgresDB.Create(work).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func GetSubmissionByWorkID(work *model.Work) error {
	result := postgresDB.Where("work_id = ?", work.WorkID).First(work)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func FindWorksByWorkTitle(title string) ([]model.Work, error) {
	var works []model.Work
	err := postgresDB.Where("work_title = ?", title).Find(&works).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return works, nil
}

func GetWorksByTrackID(trackID int) ([]model.Work, error) {
	var works []model.Work
	err := postgresDB.Where("track_id = ?", trackID).Find(&works).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return works, nil
}

func GetWorksByAuthorID(authorID int) ([]model.Work, error) {
	var works []model.Work
	err := postgresDB.Where("author_id = ?", authorID).Find(&works).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return works, nil
}

func UpdateWork(work *model.Work) error {
	err := postgresDB.Save(work).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func DeleteWork(work *model.Work) error {
	err := postgresDB.Delete(work).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func CountWorksByAuthorAndTrack(authorID int, trackID int) (int64, error) {
	var count int64
	err := postgresDB.Model(&model.Work{}).
		Where("author_id = ? AND track_id = ?", authorID, trackID).
		Count(&count).Error
	if err != nil {
		return 0, uerr.NewError(err)
	}

	return count, nil
}

func PatchWorkInfos(workID int, patch map[string]any) error {
	if len(patch) == 0 {
		return nil
	}

	var work model.Work
	err := postgresDB.Where("work_id = ?", workID).First(&work).Error
	if err != nil {
		return uerr.NewError(err)
	}

	if work.WorkInfos == nil {
		work.WorkInfos = map[string]interface{}{}
	}

	for k, v := range patch {
		work.WorkInfos[k] = v
	}

	err = postgresDB.Model(&model.Work{}).
		Where("work_id = ?", workID).
		Update("work_infos", datatypes.JSONMap(work.WorkInfos)).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func getSubmissionByID(workID int) (model.Work, error) {
	var work model.Work
	err := postgresDB.Where("work_id = ?", workID).First(&work).Error
	if err != nil {
		return model.Work{}, uerr.NewError(err)
	}

	return work, nil
}
