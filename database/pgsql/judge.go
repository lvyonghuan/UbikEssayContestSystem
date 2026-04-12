package pgsql

import (
	"main/model"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

func GetJudgesByJudgeID(judge model.Judge) error {
	err := postgresDB.Where("judge_id = ?", judge.JudgeID).First(&judge).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func CreateJudges(judge model.Judge) error {
	err := postgresDB.Create(&judge).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func UpdateJudges(judge model.Judge) error {
	err := postgresDB.Save(&judge).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func DeleteJudges(judge model.Judge) error {
	err := postgresDB.Where("judge_id = ?", judge.JudgeID).Delete(&model.Judge{}).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func GetAllJudges() ([]model.Judge, error) {
	var judges []model.Judge
	err := postgresDB.Find(&judges).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return judges, nil
}
