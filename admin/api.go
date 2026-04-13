package admin

import (
	"errors"
	"main/conf"
	_ "main/docs/API/Admin"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/response"
	"main/util/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	checkTokenFn        = token.CheckToken
	checkRefreshTokenFn = token.CheckRefreshToken
	runServerFn         = func(r *gin.Engine, port string) error {
		return r.Run(":" + port)
	}

	loginSrcFn         = loginSrc
	refreshTokenSrcFn  = refreshTokenSrc
	createContestSrcFn = createContestSrc
	updateContestSrcFn = updateContestSrc
	deleteContestSrcFn = deleteContestSrc
	createTrackSrcFn   = createTrackSrc
	updateTrackSrcFn   = updateTrackSrc
	deleteTrackSrcFn   = deleteTrackSrc
	listAuthorsSrcFn   = listAuthorsSrc
	getAuthorByIDSrcFn = getAuthorByIDSrc
	updateAuthorSrcFn  = updateAuthorSrc
	deleteAuthorSrcFn  = deleteAuthorSrc

	getWorkByIDSrcFn     = getWorkByIDSrc
	getWorkFilePathSrcFn = getWorkFilePathSrc
	queryWorksSrcFn      = queryWorksSrc
	deleteWorkSrcFn      = deleteWorkSrc

	checkAdminActiveSrcFn       = checkAdminActiveSrc
	hasPermissionSrcFn          = hasPermissionSrc
	isSuperAdminSrcFn           = isSuperAdminSrc
	createSubAdminSrcFn         = createSubAdminSrc
	batchCreateSubAdminsSrcFn   = batchCreateSubAdminsSrc
	listSubAdminsSrcFn          = listSubAdminsSrc
	updateSubAdminPermissionsFn = updateSubAdminPermissionsSrc
	disableSubAdminSrcFn        = disableSubAdminSrc
	deleteSubAdminSrcFn         = deleteSubAdminSrc
	handoverSuperAdminSrcFn     = handoverSuperAdminSrc
)

// InitRouter 初始化管理后台路由
// @title           管理后台 API
// @version         1.0
// @description     Ubik 系统管理后台服务接口文档
// @host            localhost:8081
// @BasePath        /api/v1
func InitRouter(conf conf.APIConfig) {
	r := BuildAdminRouter()
	_ = runServerFn(r, conf.AdminPort)
}

func BuildAdminRouter() *gin.Engine {
	return buildAdminRouter()
}

func buildAdminRouter() *gin.Engine {
	r := gin.Default()

	// 挂载swagger路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("Admin")))

	// 设置路由项
	v1 := r.Group("/api/v1")
	{
		admin := v1.Group("/admin")
		{
			admin.POST("/login", login)
			admin.POST("/refresh", refreshToken)

			contests := admin.Group("/contest", checkAccessToken)
			{
				contests.POST("", requirePermission(_const.PermContestCreate), createContest)
				contest := contests.Group("/:contest_id")
				{
					contest.PUT("", requirePermission(_const.PermContestUpdate), updateContest)
					contest.DELETE("", requirePermission(_const.PermContestDelete), deleteContest)
				}
			}

			tracks := admin.Group("/track", checkAccessToken)
			{
				tracks.POST("", requirePermission(_const.PermTrackCreate), createTrack)
				track := tracks.Group("/:track_id")
				{
					track.PUT("", requirePermission(_const.PermTrackUpdate), updateTrack)
					track.DELETE("", requirePermission(_const.PermTrackDelete), deleteTrack)
				}
			}

			authors := admin.Group("/authors", checkAccessToken)
			{
				authors.GET("", requirePermission(_const.PermAuthorRead), listAuthors)
				author := authors.Group("/:author_id")
				{
					author.GET("", requirePermission(_const.PermAuthorRead), getAuthorByID)
					author.PUT("", requirePermission(_const.PermAuthorUpdate), updateAuthor)
					author.DELETE("", requirePermission(_const.PermAuthorDelete), deleteAuthor)
				}
			}

			works := admin.Group("/works", checkAccessToken)
			{
				works.GET("", requirePermission(_const.PermWorksRead), getWorks)
				works.GET("/:work_id", requirePermission(_const.PermWorksRead), getWorkByID)
				works.GET("/:work_id/file", requirePermission(_const.PermWorksRead), getWorkFile)
				works.DELETE("/:work_id", requirePermission(_const.PermWorksDelete), deleteWork)
			}

			registerScriptRoutes(admin)

			subAdmins := admin.Group("/sub-admins", checkAccessToken, requireSuperAdmin())
			{
				subAdmins.GET("", listSubAdmins)
				subAdmins.POST("", createSubAdmin)
				subAdmins.POST("/batch", batchCreateSubAdmins)
				subAdmin := subAdmins.Group("/:admin_id")
				{
					subAdmin.PUT("/permissions", updateSubAdminPermissions)
					subAdmin.POST("/disable", disableSubAdmin)
					subAdmin.DELETE("", deleteSubAdmin)
				}
				subAdmins.POST("/handover-super", handoverSuperAdmin)
			}

			registerJudgeReviewRoutes(admin)
		}
	}

	return r
}

// @Summary 获取作者列表
// @Description 管理员按可选作者名查询作者列表
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param author_name query string false "作者名，可选"
// @Param offset query int false "偏移量，默认0"
// @Param limit query int false "返回条数，默认20，最大100"
// @Success 200 {object} model.Response{msg=[]model.Author} "获取成功"
// @Router /admin/authors [get]
func listAuthors(c *gin.Context) {
	authorName := strings.TrimSpace(c.Query("author_name"))

	offset := 0
	if offsetStr := strings.TrimSpace(c.DefaultQuery("offset", "0")); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil || parsedOffset < 0 {
			response.RespError(c, 400, "error: Invalid offset")
			return
		}
		offset = parsedOffset
	}

	limit := 20
	if limitStr := strings.TrimSpace(c.DefaultQuery("limit", "20")); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			response.RespError(c, 400, "error: Invalid limit")
			return
		}
		limit = parsedLimit
	}

	authors, err := listAuthorsSrcFn(authorName, offset, limit)
	if err != nil {
		response.RespError(c, 500, "error: List authors error")
		return
	}

	response.RespSuccess(c, authors)
}

// @Summary 获取作者详情
// @Description 管理员根据作者ID获取作者详细信息
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param author_id path int true "作者ID"
// @Success 200 {object} model.Response{msg=model.Author} "获取成功"
// @Router /admin/authors/{author_id} [get]
func getAuthorByID(c *gin.Context) {
	authorID, err := strconv.Atoi(c.Param("author_id"))
	if err != nil || authorID <= 0 {
		response.RespError(c, 400, "error: Invalid author_id")
		return
	}

	author, err := getAuthorByIDSrcFn(authorID)
	if err != nil {
		if errors.Is(err, errAuthorNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		response.RespError(c, 500, "error: Get author error")
		return
	}

	response.RespSuccess(c, author)
}

// @Summary 更新作者
// @Description 管理员更新指定ID的作者信息
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param author_id path int true "作者ID"
// @Param author body model.Author true "更新后的作者信息"
// @Success 200 {object} model.Response{msg=model.Author} "更新成功"
// @Router /admin/authors/{author_id} [put]
func updateAuthor(c *gin.Context) {
	authorID, err := strconv.Atoi(c.Param("author_id"))
	if err != nil || authorID <= 0 {
		response.RespError(c, 400, "error: Invalid author_id")
		return
	}

	var author model.Author
	err = c.BindJSON(&author)
	if err != nil {
		log.Logger.Warn("Update author bind json error: " + err.Error())
		response.RespError(c, 500, "error: Update author bind json error")
		return
	}

	updated, err := updateAuthorSrcFn(c.GetInt("admin_token_id"), authorID, &author)
	if err != nil {
		if errors.Is(err, errAuthorNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		response.RespError(c, 500, "error: Update author error")
		return
	}

	response.RespSuccess(c, updated)
}

// @Summary 删除作者
// @Description 管理员删除指定ID的作者
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param author_id path int true "作者ID"
// @Success 200 {object} model.Response{} "删除成功"
// @Router /admin/authors/{author_id} [delete]
func deleteAuthor(c *gin.Context) {
	authorID, err := strconv.Atoi(c.Param("author_id"))
	if err != nil || authorID <= 0 {
		response.RespError(c, 400, "error: Invalid author_id")
		return
	}

	err = deleteAuthorSrcFn(c.GetInt("admin_token_id"), authorID)
	if err != nil {
		if errors.Is(err, errAuthorNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		response.RespError(c, 500, "error: Delete author error")
		return
	}

	response.RespSuccess(c, nil)
}

// 中间件  ------------------------------------------

func checkAccessToken(c *gin.Context) {
	bearerToken := c.GetHeader("Authorization")

	id, role, err := checkTokenFn(bearerToken)
	if err != nil {
		response.RespError(c, 401, err.Error())
		c.Abort()
		return
	}

	if role != _const.RoleAdmin {
		response.RespError(c, 403, "forbidden: insufficient permissions")
		c.Abort()
		return
	}

	isActive, err := checkAdminActiveSrcFn(int(id))
	if err != nil {
		response.RespError(c, 500, "error: check admin active status failed")
		c.Abort()
		return
	}
	if !isActive {
		response.RespError(c, 403, "forbidden: admin account is disabled")
		c.Abort()
		return
	}

	c.Set("admin_token_id", int(id))
	c.Next()
}

func requirePermission(permissionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt("admin_token_id")
		hasPermission, err := hasPermissionSrcFn(adminID, permissionName)
		if err != nil {
			response.RespError(c, 500, "error: permission check failed")
			c.Abort()
			return
		}
		if !hasPermission {
			response.RespError(c, 403, "forbidden: insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

func requireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminID := c.GetInt("admin_token_id")
		isSuper, err := isSuperAdminSrcFn(adminID)
		if err != nil {
			response.RespError(c, 500, "error: super admin check failed")
			c.Abort()
			return
		}
		if !isSuper {
			response.RespError(c, 403, "forbidden: super admin required")
			c.Abort()
			return
		}

		c.Next()
	}
}

// @Summary 统一查询作品列表
// @Description 管理员按可选条件查询作品；不传某个参数表示不按该条件过滤
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param track_id query int false "赛道ID，可选"
// @Param status query string false "作品状态筛选，可选"
// @Param work_title query string false "作品名，可选"
// @Param author_name query string false "作者名，可选"
// @Param offset query int false "偏移量，默认0"
// @Param limit query int false "返回条数，默认20，最大100"
// @Success 200 {object} model.Response{msg=[]model.Work} "获取成功"
// @Router /admin/works [get]
func getWorks(c *gin.Context) {
	var trackIDPtr *int
	if trackIDStr := strings.TrimSpace(c.Query("track_id")); trackIDStr != "" {
		trackID, err := strconv.Atoi(trackIDStr)
		if err != nil || trackID <= 0 {
			response.RespError(c, 400, "error: Invalid track_id")
			return
		}
		trackIDPtr = &trackID
	}

	workStatus := strings.TrimSpace(c.Query("status"))
	workTitle := strings.TrimSpace(c.Query("work_title"))
	authorName := strings.TrimSpace(c.Query("author_name"))

	offset := 0
	if offsetStr := strings.TrimSpace(c.DefaultQuery("offset", "0")); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil || parsedOffset < 0 {
			response.RespError(c, 400, "error: Invalid offset")
			return
		}
		offset = parsedOffset
	}

	limit := 20
	if limitStr := strings.TrimSpace(c.DefaultQuery("limit", "20")); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			response.RespError(c, 400, "error: Invalid limit")
			return
		}
		limit = parsedLimit
	}

	works, err := queryWorksSrcFn(trackIDPtr, workStatus, workTitle, authorName, offset, limit)
	if err != nil {
		response.RespError(c, 500, "error: Query works error")
		return
	}

	response.RespSuccess(c, works)
}

//API handler ------------------------------------------

// 登录
// @Summary 管理员后台登录
// @Description 管理员使用用户名和密码登录，成功后返回JWT Token和Refresh Token
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param admin body model.Admin true "管理员登录信息"
// @Success 200 {object} model.Response{msg=token.ResponseToken} "登录成功"
// @Router /admin/login [post]
func login(c *gin.Context) {
	var admin model.Admin
	err := c.BindJSON(&admin)
	if err != nil {
		log.Logger.Warn("Admin login bind json error: " + err.Error())
		response.RespError(c, 500, "error: Admin login bind json error")
		return
	}

	tokens, err := loginSrcFn(admin)
	if err != nil {
		log.Logger.Warn("Admin login error: " + err.Error())
		response.RespError(c, 500, "error: Admin login error")
		return
	}

	response.RespSuccess(c, tokens)
}

// 刷新token
// @Summary 刷新管理员JWT Token
// @Description 管理员使用Refresh Token刷新JWT Token，成功后返回新的JWT Token和Refresh Token
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {refresh_token}"
// @Success 200 {object} model.Response{msg=token.ResponseToken} "刷新成功"
// @Router /admin/refresh [post]
func refreshToken(c *gin.Context) {
	bearerToken := c.GetHeader("Authorization")

	id, role, err := checkRefreshTokenFn(bearerToken)
	if err != nil {
		response.RespError(c, 401, err.Error())
		c.Abort()
		return
	}

	if role != _const.RoleAdmin {
		response.RespError(c, 403, "forbidden: insufficient permissions")
		c.Abort()
		return
	}

	isActive, activeErr := checkAdminActiveSrcFn(int(id))
	if activeErr != nil {
		response.RespError(c, 500, "error: check admin active status failed")
		c.Abort()
		return
	}
	if !isActive {
		response.RespError(c, 403, "forbidden: admin account is disabled")
		c.Abort()
		return
	}

	tokens, err := refreshTokenSrcFn(id)
	if err != nil {
		log.Logger.Warn("Admin refresh token error: " + err.Error())
		response.RespError(c, 500, "error: Admin refresh token error")
		return
	}

	response.RespSuccess(c, tokens)
}

// 创建征文活动
// @Summary 创建征文活动
// @Description 管理员创建新的征文活动
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param contest body model.Contest true "征文活动信息"
// @Success 200 {object} model.Response{msg=model.Contest} "活动创建成功"
// @Router /admin/contest [post]
func createContest(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")

	var contest model.Contest
	err := c.BindJSON(&contest)
	if err != nil {
		log.Logger.Warn("Create contest bind json error: " + err.Error())
		response.RespError(c, 500, "error: Create contest bind json error")
		return
	}

	err = createContestSrcFn(adminID, &contest)
	if err != nil {
		response.RespError(c, 500, "error: Create contest error")
		return
	}

	response.RespSuccess(c, contest)
}

// @Summary 更新征文活动
// @Description 管理员更新指定ID的征文活动
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param contest_id path int true "征文活动ID"
// @Param contest body model.Contest true "更新后的征文活动信息"
// @Success 200 {object} model.Response{msg=model.Contest} "活动更新成功"
// @Router /admin/contest/{contest_id} [put]
func updateContest(c *gin.Context) {
	contestIDStr := c.Param("contest_id")
	contestID, err := strconv.Atoi(contestIDStr)
	if err != nil {
		log.Logger.Warn("Update contest parse contest_id error: " + err.Error())
		response.RespError(c, 400, "error: Invalid contest_id")
		return
	}

	var contest model.Contest
	err = c.BindJSON(&contest)
	if err != nil {
		log.Logger.Warn("Update contest bind json error: " + err.Error())
		response.RespError(c, 500, "error: Update contest bind json error")
		return
	}

	err = updateContestSrcFn(c.GetInt("admin_token_id"), contestID, &contest)
	if err != nil {
		response.RespError(c, 500, "error: Update contest error")
		return
	}

	response.RespSuccess(c, contest)
}

// @Summary 删除征文活动
// @Description 管理员删除指定ID的征文活动
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param contest_id path int true "征文活动ID"
// @Success 200 {object} model.Response{} "活动删除成功"
// @Router /admin/contest/{contest_id} [delete]
func deleteContest(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")

	contestIDStr := c.Param("contest_id")
	contestID, err := strconv.Atoi(contestIDStr)
	if err != nil {
		log.Logger.Warn("Delete contest parse contest_id error: " + err.Error())
		response.RespError(c, 400, "error: Invalid contest_id")
		return
	}

	err = deleteContestSrcFn(adminID, contestID)
	if err != nil {
		response.RespError(c, 500, "error: Delete contest error")
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 创建赛道
// @Description 管理员创建新的赛道
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param track body model.Track true "赛道信息"
// @Success 200 {object} model.Response{msg=model.Track} "赛道创建成功"
// @Router /admin/track [post]
func createTrack(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")

	var track model.Track
	err := c.BindJSON(&track)
	if err != nil {
		log.Logger.Warn("Create track bind json error: " + err.Error())
		response.RespError(c, 500, "error: Create track bind json error")
		return
	}

	err = createTrackSrcFn(adminID, &track)
	if err != nil {
		response.RespError(c, 500, "error: Create track error")
		return
	}

	response.RespSuccess(c, track)
}

// @Summary 更新赛道
// @Description 管理员更新指定ID的赛道
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param track_id path int true "赛道ID"
// @Param track body model.Track true "更新后的赛道信息"
// @Success 200 {object} model.Response{msg=model.Track} "赛道更新成功"
// @Router /admin/track/{track_id} [put]
func updateTrack(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")

	trackIDStr := c.Param("track_id")
	trackID, err := strconv.Atoi(trackIDStr)
	if err != nil {
		log.Logger.Warn("Update track parse track_id error: " + err.Error())
		response.RespError(c, 400, "error: Invalid track_id")
		return
	}

	var track model.Track
	err = c.BindJSON(&track)
	if err != nil {
		log.Logger.Warn("Update track bind json error: " + err.Error())
		response.RespError(c, 500, "error: Update track bind json error")
		return
	}

	err = updateTrackSrcFn(adminID, trackID, &track)
	if err != nil {
		response.RespError(c, 500, "error: Update track error")
		return
	}

	response.RespSuccess(c, track)
}

// @Summary 删除赛道
// @Description 管理员删除指定ID的赛道
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param track_id path int true "赛道ID"
// @Success 200 {object} model.Response{} "赛道删除成功"
// @Router /admin/track/{track_id} [delete]
func deleteTrack(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")

	trackIDStr := c.Param("track_id")
	trackID, err := strconv.Atoi(trackIDStr)
	if err != nil {
		log.Logger.Warn("Delete track parse track_id error: " + err.Error())
		response.RespError(c, 400, "error: Invalid track_id")
		return
	}

	err = deleteTrackSrcFn(adminID, trackID)
	if err != nil {
		response.RespError(c, 500, "error: Delete track error")
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 获取作品详情
// @Description 管理员根据作品ID获取作品详细信息
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param work_id path int true "作品ID"
// @Success 200 {object} model.Response{msg=model.Work} "获取成功"
// @Router /admin/works/{work_id} [get]
func getWorkByID(c *gin.Context) {
	workID, err := strconv.Atoi(c.Param("work_id"))
	if err != nil {
		response.RespError(c, 400, "error: Invalid work_id")
		return
	}

	work, err := getWorkByIDSrcFn(workID)
	if err != nil {
		if errors.Is(err, errWorkNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		response.RespError(c, 500, "error: Get work error")
		return
	}

	response.RespSuccess(c, work)
}

// @Summary 获取作品文件
// @Description 管理员根据作品ID获取作品文件
// @Tags Admin
// @Accept application/json
// @Produce application/octet-stream
// @Param Authorization header string true "Bearer {access_token}"
// @Param work_id path int true "作品ID"
// @Success 200 {file} file "作品文件"
// @Router /admin/works/{work_id}/file [get]
func getWorkFile(c *gin.Context) {
	workID, err := strconv.Atoi(c.Param("work_id"))
	if err != nil {
		response.RespError(c, 400, "error: Invalid work_id")
		return
	}

	filePath, err := getWorkFilePathSrcFn(workID)
	if err != nil {
		if errors.Is(err, errWorkNotFound) || errors.Is(err, errWorkFileNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		response.RespError(c, 500, "error: Get work file error")
		return
	}

	c.FileAttachment(filePath, filepath.Base(filePath))
}

// @Summary 删除作品
// @Description 管理员根据作品ID删除作品及其存储文件
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param work_id path int true "作品ID"
// @Success 200 {object} model.Response{} "删除成功"
// @Router /admin/works/{work_id} [delete]
func deleteWork(c *gin.Context) {
	workID, err := strconv.Atoi(c.Param("work_id"))
	if err != nil {
		response.RespError(c, 400, "error: Invalid work_id")
		return
	}

	err = deleteWorkSrcFn(c.GetInt("admin_token_id"), workID)
	if err != nil {
		if errors.Is(err, errWorkNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		response.RespError(c, 500, "error: Delete work error")
		return
	}

	response.RespSuccess(c, nil)
}
