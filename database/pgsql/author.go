package pgsql

import (
	"errors"
	"main/model"
	_const "main/util/const"
	"strings"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func GetAuthorByAuthorName(author *model.Author) error {
	result := postgresDB.Table("authors").Where("author_name = ?", author.AuthorName).First(author)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return _const.UsernameNotExist
		}

		return uerr.NewError(result.Error)
	}

	return nil
}

func GetAuthorByAuthorID(author *model.Author) error {
	result := postgresDB.Table("authors").Where("author_id = ?", author.AuthorID).First(author)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func CreateAuthor(author *model.Author) error {
	result := postgresDB.Create(author)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func UpdateAuthor(author *model.Author) error {
	result := postgresDB.Model(author).Updates(author)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func ListAuthors(authorName string, offset int, limit int) ([]model.Author, error) {
	var authors []model.Author

	query := postgresDB.Table("authors")
	if trimmed := strings.TrimSpace(authorName); trimmed != "" {
		query = query.Where("author_name = ?", trimmed)
	}

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}

	result := query.Order("author_id ASC").Offset(offset).Limit(limit).Find(&authors)
	if result.Error != nil {
		return nil, uerr.NewError(result.Error)
	}

	return authors, nil
}

func UpdateAuthorByID(authorID int, updated *model.Author) (model.Author, error) {
	var existing model.Author
	if err := postgresDB.Table("authors").Where("author_id = ?", authorID).First(&existing).Error; err != nil {
		return model.Author{}, uerr.NewError(err)
	}

	authorInfos := datatypes.JSONMap{}
	if updated.AuthorInfos != nil {
		authorInfos = datatypes.JSONMap(updated.AuthorInfos)
	}

	patch := map[string]interface{}{
		"author_name":  updated.AuthorName,
		"pen_name":     updated.PenName,
		"author_email": updated.AuthorEmail,
		"author_infos": authorInfos,
	}

	if err := postgresDB.Table("authors").Where("author_id = ?", authorID).Updates(patch).Error; err != nil {
		return model.Author{}, uerr.NewError(err)
	}

	if err := postgresDB.Table("authors").Where("author_id = ?", authorID).First(&existing).Error; err != nil {
		return model.Author{}, uerr.NewError(err)
	}

	return existing, nil
}

func DeleteAuthorByID(authorID int) (model.Author, error) {
	var author model.Author
	if err := postgresDB.Table("authors").Where("author_id = ?", authorID).First(&author).Error; err != nil {
		return model.Author{}, uerr.NewError(err)
	}

	if err := postgresDB.Table("authors").Where("author_id = ?", authorID).Delete(&model.Author{}).Error; err != nil {
		return model.Author{}, uerr.NewError(err)
	}

	return author, nil
}

func DeleteAuthor(author *model.Author) error {
	result := postgresDB.Delete(author)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}
