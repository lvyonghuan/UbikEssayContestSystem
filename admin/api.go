package admin

import (
	"main/conf"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/response"
	"main/util/token"
	"strconv"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// InitRouter 初始化管理后台路由
// @title           管理后台 API
// @version         1.0
// @description     Ubik 系统管理后台服务接口文档
// @host            localhost:8081
// @BasePath        /api/v1
func InitRouter(conf conf.APIConfig) {
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
				works.GET("/:work_id")          //获取作品详细信息
				works.GET("/:work_id/file")     //获取作品文件，按照./submissions/{track_id}/{author_id}/{work_id}.suffix的形式存储
				works.GET("/track/:track_id")   //获取指定赛道的所有作品
				works.GET("/author/:author_id") //获取指定作者的所有作品
				works.DELETE("/:work_id")       //删除指定作品（同时要删除存储）
			}
		}
	}

	r.Run(":" + conf.AdminPort)
}

// 中间件  ------------------------------------------

func checkAccessToken(c *gin.Context) {
	bearerToken := c.GetHeader("Authorization")

	id, role, err := token.CheckToken(bearerToken)
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

	tokens, err := loginSrc(admin)
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

	id, role, err := token.CheckRefreshToken(bearerToken)
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

	tokens, err := refreshTokenSrc(id)
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

	err = createContestSrc(adminID, &contest)
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

	err = updateContestSrc(c.GetInt("admin_token_id"), contestID, &contest)
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

	err = deleteContestSrc(adminID, contestID)
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

	err = createTrackSrc(adminID, &track)
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

	err = updateTrackSrc(adminID, trackID, &track)
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

	err = deleteTrackSrc(adminID, trackID)
	if err != nil {
		response.RespError(c, 500, "error: Delete track error")
		return
	}

	response.RespSuccess(c, nil)
}
