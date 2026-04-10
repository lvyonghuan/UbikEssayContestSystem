package submission

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"main/database/pgsql"
	"main/database/redis"
	"main/model"
	_const "main/util/const"
	"main/util/document"
	"main/util/log"
	"main/util/response"
	"main/util/scriptflow"
	"main/util/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lvyonghuan/Ubik-Util/uerr"
)

var (
	checkRefreshTokenFn = token.CheckRefreshToken

	registerAuthorSrcFn         = registerAuthorSrc
	authorLoginSrcFn            = authorLoginSrc
	refreshTokenSrcFn           = refreshTokenSrc
	updateAuthorSrcFn           = updateAuthorSrc
	submissionWorkSrcFn         = submissionWorkSrc
	findSubmissionsByAuthorIDFn = findSubmissionsByAuthorIDSrc
	updateSubmissionSrcFn       = updateSubmissionSrc
	deleteSubmissionSrcFn       = deleteSubmissionSrc

	getSubmissionByWorkIDFn     = pgsql.GetSubmissionByWorkID
	getUploadFilePermissionFn   = redis.GetUploadFilePermission
	runTrackHookFn              = runTrackHook
	patchWorkInfosFn            = pgsql.PatchWorkInfos
	updateWorkStatusFn          = pgsql.UpdateWorkStatus
	resolveSubmissionFilePathFn = resolveSubmissionFilePath
	computeFileSHA256Fn         = computeFileSHA256

	newDocumentConverterFn = func() document.Converter {
		return document.NewLibreOfficeConverter("", "", 60*time.Second)
	}
)

// @Summary 作者注册
// @Description 作者注册接口，接受作者信息并创建新账号
// @Tags 作者端
// @Accept application/json
// @Produce application/json
// @Param author body model.Author true "作者注册信息"
// @Success 200 {object} model.Response "注册成功"
// @Router /author/register [post]
func authorRegister(c *gin.Context) {
	var author model.Author
	err := c.ShouldBind(&author)
	if err != nil {
		log.Logger.Warn("Failed to bind author registration data: " + err.Error())
		response.RespError(c, 400, "bad request")
		return
	}

	author.AuthorEmail = strings.TrimSpace(author.AuthorEmail)
	if author.AuthorEmail == "" {
		response.RespError(c, 400, "bad request: author email is required")
		return
	}

	if strings.TrimSpace(author.PenName) == "" {
		author.PenName = author.AuthorName
	}

	err = registerAuthorSrcFn(&author)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 作者登录
// @Description 作者登录接口，返回访问令牌和刷新令牌
// @Tags 作者端
// @Accept application/json
// @Produce application/json
// @Param author body model.Author true "作者登录信息"
// @Success 200 {object} model.Response{msg=token.ResponseToken} "登录成功，返回访问令牌和刷新令牌"
// @Router /author/login [post]
func authorLogin(c *gin.Context) {
	var author model.Author
	err := c.ShouldBind(&author)
	if err != nil {
		log.Logger.Warn("Failed to bind author login data: " + err.Error())
		response.RespError(c, 400, "bad request")
		return
	}

	tokens, err := authorLoginSrcFn(&author)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, tokens)
}

// @Summary 刷新令牌
// @Description 刷新访问令牌，接受刷新令牌并返回新的访问令牌和刷新令牌
// @Tags 作者端
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {refresh_token}"
// @Success 200 {object} model.Response{msg=token.ResponseToken} "刷新成功，返回新的访问令牌和刷新令牌"
// @Router /author/refresh [get]
func refreshToken(c *gin.Context) {
	bearerToken := c.GetHeader("Authorization")

	id, role, err := checkRefreshTokenFn(bearerToken)
	if err != nil {
		response.RespError(c, 401, err.Error())
		c.Abort()
		return
	}

	if role != _const.RoleAuthor {
		response.RespError(c, 403, "forbidden: insufficient permissions")
		c.Abort()
		return
	}

	tokens, err := refreshTokenSrcFn(id)
	if err != nil {
		log.Logger.Warn("Author refresh token error: " + err.Error())
		response.RespError(c, 500, "error: Admin refresh token error")
		return
	}

	response.RespSuccess(c, tokens)
}

// @Summary 更新作者信息
// @Description 更新作者信息接口，接受新的作者信息并更新数据库中的记录
// @Tags 作者端
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param author body model.Author true "新的作者信息"
// @Success 200 {object} model.Response "更新成功"
// @Router /author [put]
func updateAuthor(c *gin.Context) {
	var author model.Author
	err := c.ShouldBind(&author)
	if err != nil {
		log.Logger.Warn("Failed to bind author update data: " + err.Error())
		response.RespError(c, 400, "bad request")
		return
	}

	tokenAuthorID := c.GetInt("author_token_id")
	if author.AuthorID != 0 && author.AuthorID != tokenAuthorID {
		response.RespError(c, 403, "forbidden: can only update your own profile")
		return
	}
	author.AuthorID = tokenAuthorID

	err = updateAuthorSrcFn(&author)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

// FIXME 处理文件后缀名
// @Summary 提交作品
// @Description 提交作品接口，接受作品信息和文件并创建新的作品记录
// @Tags 作者端
// @Accept multipart/form-data
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param work body model.Work true "作品信息"
// @Success 200 {object} model.Response{msg=model.Work} "提交成功，返回提交的作品信息"
// @Router /author/submission [post]
func submissionWork(c *gin.Context) {
	workInt, isExist := c.Get("work")
	if !isExist {
		response.RespError(c, 400, "bad request")
		return
	}
	work, isOK := workInt.(model.Work)
	if !isOK {
		response.RespError(c, 400, "bad request")
		return
	}

	err := submissionWorkSrcFn(&work)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, work)
}

// @Summary 获取提交记录
// @Description 获取提交记录接口，返回作者的所有提交记录
// @Tags 作者端
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Success 200 {object} model.Response{msg=[]model.Work} "获取成功，返回提交记录列表"
// @Router /author/submission [get]
func getSubmissions(c *gin.Context) {
	authorID := c.GetInt("author_token_id")
	works, err := findSubmissionsByAuthorIDFn(authorID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, works)
}

// @Summary 更新提交记录
// @Description 更新提交记录接口，接受新的作品信息并更新数据库中的记录
// @Tags 作者端
// @Accept multipart/form-data
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param work body model.Work true "新的作品信息"
// @Success 200 {object} model.Response{msg=model.Work} "更新成功，返回更新后的作品信息"
// @Router /author/submission [put]
func updateSubmission(c *gin.Context) {
	workInt, isExist := c.Get("work")
	if !isExist {
		response.RespError(c, 400, "bad request")
		c.Abort()
		return
	}

	work, isOK := workInt.(model.Work)
	if !isOK {
		response.RespError(c, 400, "bad request")
		c.Abort()
		return
	}

	err := updateSubmissionSrcFn(&work)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, work)
}

// @Summary 删除提交记录
// @Description 删除提交记录接口，接受作品ID并删除对应的作品记录
// @Tags 作者端
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param work body model.Work true "作品信息"
// @Success 200 {object} model.Response{msg=string} "删除成功"
// @Router /author/submission [delete]
func deleteSubmission(c *gin.Context) {
	workInt, isExist := c.Get("work")
	if !isExist {
		response.RespError(c, 400, "bad request")
		return
	}

	work, isOK := workInt.(model.Work)
	if !isOK {
		response.RespError(c, 400, "bad request")
		return
	}

	err := deleteSubmissionSrcFn(&work)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 上传提交文件
// @Description 上传提交文件接口，接受作品ID和文件并保存到服务器
// @Tags 作者端
// @Accept multipart/form-data
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param work_id formData int true "作品ID"
// @Param article_file formData file true "提交文件"
// @Param file_hash formData string true "提交文件 SHA-256（64位十六进制）"
// @Success 200 {object} model.Response "上传成功"
// @Router /author/submission/file [post]
func saveSubmissionFile(c *gin.Context) {
	workIDStr := c.PostForm("work_id")
	workIDInt, err := strconv.Atoi(workIDStr)
	if err != nil {
		response.RespError(c, 400, "bad request")
		return
	}
	authorID, trackID, err := getUploadFilePermissionFn(workIDInt)
	if err != nil {
		log.Logger.Warn("Author upload file permission error: " + err.Error())
		response.RespError(c, 500, uerr.ExtractError(err).Error())
		return
	}

	fileOrigin, err := c.FormFile("article_file")
	if err != nil {
		response.RespError(c, 400, "bad request")
		return
	}
	suffix := strings.ToLower(filepath.Ext(fileOrigin.Filename))

	switch suffix {
	case ".docx", ".doc":
	default:
		response.RespError(c, 400, "bad request: unsupported file type")
		return
	}

	clientFileHash, err := normalizeSHA256Hex(c.PostForm("file_hash"))
	if err != nil {
		response.RespError(c, 400, err.Error())
		return
	}

	preHookResult, err := runTrackHookFn(
		scriptflow.ScopeSubmission,
		scriptflow.EventFilePre,
		trackID,
		map[string]any{
			"phase":      "file_pre",
			"workID":     workIDInt,
			"authorID":   authorID,
			"trackID":    trackID,
			"fileName":   fileOrigin.Filename,
			"fileSize":   fileOrigin.Size,
			"fileSuffix": suffix,
			"fileHash":   clientFileHash,
		},
	)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	if !preHookResult.Allowed {
		reason := preHookResult.Reason
		if reason == "" {
			reason = "upload blocked by script flow"
		}
		response.RespError(c, 403, reason)
		return
	}
	if len(preHookResult.Patch) > 0 {
		if err := patchWorkInfosFn(workIDInt, preHookResult.Patch); err != nil {
			response.RespError(c, 500, err.Error())
			return
		}
		if err := applyPersistedWorkStatusPatch(workIDInt, preHookResult.Patch); err != nil {
			response.RespError(c, 500, err.Error())
			return
		}
	}

	dstDir := filepath.Join(_const.FileRootPath, strconv.Itoa(trackID), strconv.Itoa(authorID))
	if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	if err := cleanupSubmissionFileVariants(dstDir, workIDInt); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	finalDocxPath := filepath.Join(dstDir, strconv.Itoa(workIDInt)+".docx")
	if suffix == ".docx" {
		err = c.SaveUploadedFile(fileOrigin, finalDocxPath)
		if err != nil {
			response.RespError(c, 500, err.Error())
			return
		}
	} else {
		tempDocPath := filepath.Join(dstDir, strconv.Itoa(workIDInt)+".doc")
		err = c.SaveUploadedFile(fileOrigin, tempDocPath)
		if err != nil {
			response.RespError(c, 500, err.Error())
			return
		}

		converter := newDocumentConverterFn()
		err = converter.ConvertDocToDocx(c.Request.Context(), tempDocPath, finalDocxPath)
		_ = os.Remove(tempDocPath)
		if err != nil {
			response.RespError(c, 500, err.Error())
			return
		}
	}

	actualFileHash, actualFileSize, err := computeFileSHA256Fn(finalDocxPath)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	if actualFileHash != clientFileHash {
		_ = os.Remove(finalDocxPath)
		response.RespError(c, 400, "bad request: file hash mismatch")
		return
	}

	if err := patchWorkInfosFn(workIDInt, map[string]any{
		"file_hash_sha256": actualFileHash,
		"file_size_bytes":  actualFileSize,
		"file_uploaded_at": time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	postHookResult, err := runTrackHookFn(
		scriptflow.ScopeSubmission,
		scriptflow.EventFilePost,
		trackID,
		map[string]any{
			"phase":       "file_post",
			"workID":      workIDInt,
			"authorID":    authorID,
			"trackID":     trackID,
			"savedPath":   filepath.ToSlash(finalDocxPath),
			"savedSuffix": ".docx",
			"fileHash":    actualFileHash,
			"fileSize":    actualFileSize,
		},
	)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	if !postHookResult.Allowed {
		reason := postHookResult.Reason
		if reason == "" {
			reason = "upload blocked by script flow"
		}
		response.RespError(c, 403, reason)
		return
	}
	if len(postHookResult.Patch) > 0 {
		if err := patchWorkInfosFn(workIDInt, postHookResult.Patch); err != nil {
			response.RespError(c, 500, err.Error())
			return
		}
		if err := applyPersistedWorkStatusPatch(workIDInt, postHookResult.Patch); err != nil {
			response.RespError(c, 500, err.Error())
			return
		}
	}

	response.RespSuccess(c, nil)
}

// @Summary 下载提交文件
// @Description 下载当前作者自己的投稿文件
// @Tags 作者端
// @Accept application/json
// @Produce application/octet-stream
// @Param Authorization header string true "Bearer {access_token}"
// @Param submission_id path int true "投稿ID(work_id)"
// @Success 200 {file} file "投稿文件"
// @Router /author/submission/file/{submission_id} [get]
func getSubmissionFile(c *gin.Context) {
	submissionID, err := strconv.Atoi(c.Param("submission_id"))
	if err != nil || submissionID <= 0 {
		response.RespError(c, 400, "bad request")
		return
	}

	work := model.Work{WorkID: submissionID}
	if err := getSubmissionByWorkIDFn(&work); err != nil {
		if isSubmissionNotFoundError(err) {
			response.RespError(c, 404, "submission not found")
			return
		}
		response.RespError(c, 500, uerr.ExtractError(err).Error())
		return
	}

	if work.AuthorID != c.GetInt("author_token_id") {
		response.RespError(c, 403, "forbidden: can only access your own submissions")
		return
	}

	filePath, err := resolveSubmissionFilePathFn(work)
	if err != nil {
		if isSubmissionNotFoundError(err) {
			response.RespError(c, 404, "submission file not found")
			return
		}
		response.RespError(c, 500, err.Error())
		return
	}

	fileHash, _, err := computeFileSHA256Fn(filePath)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	c.Header("X-File-SHA256", fileHash)
	c.FileAttachment(filePath, filepath.Base(filePath))
}

func cleanupSubmissionFileVariants(dstDir string, workID int) error {
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	prefix := strconv.Itoa(workID) + "."
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		if err := os.Remove(filepath.Join(dstDir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

func resolveSubmissionFilePath(work model.Work) (string, error) {
	dstDir := filepath.Join(_const.FileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", os.ErrNotExist
		}
		return "", err
	}

	prefix := strconv.Itoa(work.WorkID) + "."
	selectedName := ""
	selectedTime := time.Time{}
	hasDocx := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		isDocx := ext == ".docx"

		if isDocx {
			if !hasDocx || selectedName == "" || info.ModTime().After(selectedTime) {
				hasDocx = true
				selectedName = name
				selectedTime = info.ModTime()
			}
			continue
		}

		if hasDocx {
			continue
		}

		if selectedName == "" || info.ModTime().After(selectedTime) {
			selectedName = name
			selectedTime = info.ModTime()
		}
	}

	if selectedName == "" {
		return "", os.ErrNotExist
	}

	return filepath.Join(dstDir, selectedName), nil
}

func computeFileSHA256(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		_ = file.Close()
	}()

	hashBuilder := sha256.New()
	size, err := io.Copy(hashBuilder, file)
	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(hashBuilder.Sum(nil)), size, nil
}

func normalizeSHA256Hex(value string) (string, error) {
	hash := strings.ToLower(strings.TrimSpace(value))
	if len(hash) != 64 {
		return "", errors.New("bad request: file_hash must be 64 hex characters")
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return "", errors.New("bad request: file_hash must be 64 hex characters")
	}
	return hash, nil
}

func applyPersistedWorkStatusPatch(workID int, patch map[string]any) error {
	status, ok := extractWorkStatusFromPatch(patch)
	if !ok {
		return nil
	}

	if err := updateWorkStatusFn(workID, status); err != nil {
		return uerr.ExtractError(err)
	}

	return nil
}

func isSubmissionNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	text := strings.ToLower(uerr.ExtractError(err).Error())
	return strings.Contains(text, "record not found") || strings.Contains(text, "not found")
}
