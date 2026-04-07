package submission

import (
	"errors"
	"main/database/pgsql"
	"main/database/redis"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/password"
	"main/util/token"
	"os"
	"strconv"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

func registerAuthorSrc(author *model.Author) error {
	tmpAuthor := model.Author{
		AuthorName: author.AuthorName,
	}

	//查询用户名是否已经被注册过
	err := pgsql.GetAuthorByAuthorName(&tmpAuthor)
	if err != nil {
		if errors.Is(err, _const.UsernameNotExist) {
			err := pgsql.CreateAuthor(author)
			if err != nil {
				log.Logger.Warn("Register author failed: " + err.Error())
				return uerr.ExtractError(err)
			}
		} else {
			log.Logger.Warn("Failed to check if username exists: " + err.Error())
			return uerr.ExtractError(err)
		}
	} else {
		return errors.New("username already exists")
	}

	return nil
}

func authorLoginSrc(author *model.Author) (token.ResponseToken, error) {
	tempAuthor := model.Author{
		AuthorID: author.AuthorID,
	}

	err := pgsql.GetAuthorByAuthorID(&tempAuthor)
	if err != nil {
		log.Logger.Warn("Author login failed: " + err.Error())
		return token.ResponseToken{}, uerr.ExtractError(err)
	}

	if !password.CheckPasswordHash(author.Password, tempAuthor.Password) {
		return token.ResponseToken{}, errors.New("bad request")
	}

	//生成token
	tokens, err := token.GenTokenAndRefreshToken(int64(tempAuthor.AuthorID), _const.RoleAuthor)
	if err != nil {
		log.Logger.Warn("Generate token failed: " + err.Error())
		return token.ResponseToken{}, uerr.ExtractError(err)
	}

	return tokens, nil
}

func refreshTokenSrc(adminID int64) (token.ResponseToken, error) {
	return token.GenTokenAndRefreshToken(adminID, _const.RoleAdmin)
}

func updateAuthorSrc(author *model.Author) error {
	err := pgsql.UpdateAuthor(author)
	if err != nil {
		log.Logger.Warn("Update author failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	return nil
}

//func deleteAuthorSrc(author *model.Author) error {
//	err := pgsql.DeleteAuthor(author)
//	if err != nil {
//		log.Logger.Warn("Delete author failed: " + err.Error())
//		return uerr.ExtractError(err)
//	}
//
//	return nil
//}

func submissionWorkSrc(work *model.Work) error {
	err := checkSubmissionTimeValid(work.TrackID)
	if err != nil {
		return err
	}

	err = pgsql.SubmissionWork(work)
	if err != nil {
		log.Logger.Warn("Submission work failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	//设置redis缓存
	err = redis.SetUploadFilePermission(work.AuthorID, work.TrackID, work.WorkID)
	if err != nil {
		log.Logger.Warn("Set upload file permission failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	return nil
}

func updateSubmissionSrc(work *model.Work) error {
	//检查投稿日期是否已经开始或已经结束
	if err := checkSubmissionTimeValid(work.TrackID); err != nil {
		return err
	}

	err := pgsql.UpdateWork(work)
	if err != nil {
		log.Logger.Warn("Update submission failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	//设置redis缓存
	err = redis.SetUploadFilePermission(work.AuthorID, work.TrackID, work.WorkID)
	if err != nil {
		log.Logger.Warn("Set upload file permission failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	return nil
}

func deleteSubmissionSrc(work *model.Work) error {
	err := pgsql.DeleteWork(work)
	if err != nil {
		log.Logger.Warn("Delete submission failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	dstDir := _const.FileRootPath + "/" + strconv.Itoa(work.TrackID) + "/" + strconv.Itoa(work.AuthorID)
	prefix := strconv.Itoa(work.WorkID)
	// 遍历目录，删除以 workID. 开头的文件（忽略后缀）
	// FIXME 修改文件后缀处理逻辑
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		log.Logger.Warn("Delete submission failed: " + err.Error())
		return nil
	}
	for _, e := range entries {
		name := e.Name()
		if len(name) > len(prefix) && name[:len(prefix)+1] == prefix+"." {
			if err := os.Remove(dstDir + "/" + name); err != nil {
				log.Logger.Warn("Delete submission failed: " + err.Error())
			}
		}
	}

	return nil
}

func findSubmissionsByAuthorIDSrc(authorID int) ([]model.Work, error) {
	works, err := pgsql.GetWorksByAuthorID(authorID)
	if err != nil {
		log.Logger.Warn("Failed to get works by author id: " + err.Error())
		return nil, uerr.ExtractError(err)
	}
	return works, nil
}

// 检查投稿日期是否符合要求
func checkSubmissionTimeValid(trackID int) error {
	//检查投稿日期是否已经开始或已经结束
	start, end, err := redis.GetStartAndEndDate(trackID)
	if err != nil {
		return uerr.ExtractError(err)
	}
	nowTimeUnix := time.Now().Unix()
	switch {
	case nowTimeUnix < start:
		return errors.New("contest has not started yet")
	case nowTimeUnix > end:
		return errors.New("contest has already ended")

	default:
		return nil
	}
}
