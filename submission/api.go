package submission

import (
	"main/database/redis"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/response"
	"main/util/token"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lvyonghuan/Ubik-Util/uerr"
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

	err = registerAuthorSrc(&author)
	if err != nil {
		response.RespError(c, 500, err.Error())
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

	tokens, err := authorLoginSrc(&author)
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

	id, role, err := token.CheckRefreshToken(bearerToken)
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

	tokens, err := refreshTokenSrc(id)
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

	if author.AuthorID != c.GetInt("author_token_id") {
		response.RespError(c, 403, "Forbidden")
		return
	}

	err = updateAuthorSrc(&author)
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
// @Router /submission [post]
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

	err := submissionWorkSrc(&work)
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
// @Param id path int true "作者ID"
// @Success 200 {object} model.Response{msg=[]model.Work} "获取成功，返回提交记录列表"
// @Router /submission/{id} [get]
func getSubmissions(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if id != c.GetInt("author_token_id") {
		response.RespError(c, 403, "Forbidden")
		return
	}

	works, err := findSubmissionsByAuthorIDSrc(id)
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
// @Router /submission [put]
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

	err := updateSubmissionSrc(&work)
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
// @Router /submission [delete]
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

	err := deleteSubmissionSrc(&work)
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
// @Success 200 {object} model.Response "上传成功"
// @Router /submission/file [post]
func saveSubmissionFile(c *gin.Context) {
	workID := c.PostForm("work_id")

	//1.校验文档是否已经在数据库提交
	workIDInt, err := strconv.Atoi(workID)
	if err != nil {
		response.RespError(c, 400, "bad request")
		return
	}
	authorID, trackID, err := redis.GetUploadFilePermission(workIDInt)
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
	suffix := fileOrigin.Filename[strings.LastIndex(fileOrigin.Filename, "."):]

	//检查后缀名是否是允许的类型
	switch suffix {
	case "docx":
	case "doc":
	//TODO 转换为docx
	default:
		response.RespError(c, 400, "bad request: unsupported file type")
		return
	}

	dst := _const.FileRootPath + "/" + strconv.Itoa(trackID) + "/" + strconv.Itoa(authorID) + "/" + strconv.Itoa(workIDInt) + suffix
	if err := os.MkdirAll(_const.FileRootPath+"/"+strconv.Itoa(trackID)+"/"+strconv.Itoa(authorID), os.ModePerm); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	err = c.SaveUploadedFile(fileOrigin, dst)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}
