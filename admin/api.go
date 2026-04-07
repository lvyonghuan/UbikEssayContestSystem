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

	getWorkByIDSrcFn        = getWorkByIDSrc
	getWorkFilePathSrcFn    = getWorkFilePathSrc
	getWorksByTrackIDSrcFn  = getWorksByTrackIDSrc
	getWorksByAuthorIDSrcFn = getWorksByAuthorIDSrc
	deleteWorkSrcFn         = deleteWorkSrc
)

// InitRouter 初始化管理后台路由
// @title           管理后台 API
// @version         1.0
// @description     Ubik 系统管理后台服务接口文档
// @host            localhost:8081
// @BasePath        /api/v1
func InitRouter(conf conf.APIConfig) {
	r := buildAdminRouter()
	_ = runServerFn(r, conf.AdminPort)
}

func buildAdminRouter() *gin.Engine {
	r := gin.Default()

	// 挂载swagger路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("Admin")))

	//设置路由项
	v1 := r.Group("/api/v1")
	{
		admin := v1.Group("/admin")
		{
			admin.POST("/login", login)          //admin账号只能由超级管理员衍生，不能注册
			admin.POST("/refresh", refreshToken) //刷新token

			contests := admin.Group("/contest", checkAccessToken)
			{
				contests.POST("", createContest) //创建征文活动
				contest := contests.Group("/:contest_id")
				{
					contest.PUT("", updateContest)    //更新征文活动
					contest.DELETE("", deleteContest) //删除征文活动
				}
			}

			tracks := admin.Group("/track", checkAccessToken)
			{
				tracks.POST("", createTrack) //创建赛道
				track := tracks.Group("/:track_id")
				{
					track.PUT("", updateTrack)    //更新赛道
					track.DELETE("", deleteTrack) //删除赛道
				}
			}

			works := admin.Group("/works", checkAccessToken)
			{
				works.GET("/:work_id", getWorkByID)                 //获取作品详细信息
				works.GET("/:work_id/file", getWorkFile)            //获取作品文件，按照./submissions/{track_id}/{author_id}/{work_id}.suffix的形式存储
				works.GET("/track/:track_id", getWorksByTrackID)    //获取指定赛道的所有作品
				works.GET("/author/:author_id", getWorksByAuthorID) //获取指定作者的所有作品
				works.DELETE("/:work_id", deleteWork)               //删除指定作品（同时要删除存储）
			}
		}
	}

	return r
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

	c.Set("admin_token_id", int(id))
	c.Next()
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

// @Summary 按赛道获取作品列表
// @Description 管理员根据赛道ID获取该赛道下的全部作品
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param track_id path int true "赛道ID"
// @Success 200 {object} model.Response{msg=[]model.Work} "获取成功"
// @Router /admin/works/track/{track_id} [get]
func getWorksByTrackID(c *gin.Context) {
	trackID, err := strconv.Atoi(c.Param("track_id"))
	if err != nil {
		response.RespError(c, 400, "error: Invalid track_id")
		return
	}

	works, err := getWorksByTrackIDSrcFn(trackID)
	if err != nil {
		response.RespError(c, 500, "error: Get works by track_id error")
		return
	}

	response.RespSuccess(c, works)
}

// @Summary 按作者获取作品列表
// @Description 管理员根据作者ID获取该作者的全部作品
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param author_id path int true "作者ID"
// @Success 200 {object} model.Response{msg=[]model.Work} "获取成功"
// @Router /admin/works/author/{author_id} [get]
func getWorksByAuthorID(c *gin.Context) {
	authorID, err := strconv.Atoi(c.Param("author_id"))
	if err != nil {
		response.RespError(c, 400, "error: Invalid author_id")
		return
	}

	works, err := getWorksByAuthorIDSrcFn(authorID)
	if err != nil {
		response.RespError(c, 500, "error: Get works by author_id error")
		return
	}

	response.RespSuccess(c, works)
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
