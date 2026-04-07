package pgsql

import (
	"errors"
	"main/model"
	_const "main/util/const"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

func GetAuthorByAuthorName(author *model.Author) error {
	result := postgresDB.Table("author").Where("author_name = ?", author.AuthorName).First(author)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return _const.UsernameNotExist
		}

		return uerr.NewError(result.Error)
	}

	return nil
}

func GetAuthorByAuthorID(author *model.Author) error {
	result := postgresDB.Table("author").Where("author_id = ?", author.AuthorID).First(author)
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

func DeleteAuthor(author *model.Author) error {
	result := postgresDB.Delete(author)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}
