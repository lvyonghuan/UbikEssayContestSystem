package pgsql

import (
	"main/model"

	"github.com/lvyonghuan/Ubik-Util/uerr"
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

func getSubmissionByID(workID int) (model.Work, error) {
	var work model.Work
	err := postgresDB.Where("work_id = ?", workID).First(&work).Error
	if err != nil {
		return model.Work{}, uerr.NewError(err)
	}

	return work, nil
}
