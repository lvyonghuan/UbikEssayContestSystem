package system

import (
	"errors"
	"main/conf"
	_ "main/docs/API/System"
	"main/util/log"
	"main/util/response"
	"main/util/token"
	"strconv"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	checkTokenFn = token.CheckToken
	runServerFn  = func(r *gin.Engine, port string) error {
		return r.Run(":" + port)
	}

	getContestSrcFn   = getContestSrc
	getContestByIDFn  = getContestByIDSrc
	getTracksSrcFn    = getTracksSrc
	getTrackByIDSrcFn = getTrackByIDSrc
)

// initGlobalInfoRouter 只能有GET
// @title           全局信息 API
// @version         1.0
// @description     Ubik 系统全局信息接口文档
// @host            localhost:8082
// @BasePath        /api/v1
func initGlobalInfoRouter(apiConf conf.APIConfig) {
	r := buildGlobalInfoRouter()
	_ = runServerFn(r, apiConf.GlobalInfoPort)
}

func buildGlobalInfoRouter() *gin.Engine {
	r := gin.Default()

	// 挂载swagger路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("GlobalInfo")))

	//这里的所有操作都需要先check一个合法token，避免ddos
	v1 := r.Group("/api/v1", checkAccessToken)
	{
		contests := v1.Group("/contests")
		{
			contests.GET("", getContests) //获取赛事列表
			contests.GET("/:contest_id", getContestByID)
		}

		tracks := v1.Group("/tracks")
		{
			tracks.GET("/:contest_id", getTracks) // 获取赛事下赛道列表
			tracks.GET("/detail/:track_id", getTrackByID)
		}
	}

	return r
}

// --- 中间件
func checkAccessToken(c *gin.Context) {
	bearerToken := c.GetHeader("Authorization")
	_, _, err := checkTokenFn(bearerToken)
	if err != nil {
		log.Logger.Warn("访问全局信息接口时发生授权错误: " + err.Error())
		response.RespError(c, 401, "Unauthorized: Invalid token")
		c.Abort()
		return
	}
}

// @Summary 获取赛事列表
// @Description 获取系统中所有的赛事列表，包括基本信息如赛事ID、名称、状态等
// @Tags Contests
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Success 200 {object} model.Response{msg=[]model.Contest} "成功反回赛事列表"
// @Router /contests [get]
func getContests(c *gin.Context) {
	contests, err := getContestSrcFn()
	if err != nil {
		log.Logger.Warn("Failed to get contest src: " + err.Error())
		response.RespError(c, 500, "Failed to get contests")
		return
	}

	response.RespSuccess(c, contests)
}

// @Summary 获取赛事详情
// @Description 获取指定赛事详情，包括基础信息与时间范围
// @Tags Contests
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param contest_id path int true "赛事ID"
// @Success 200 {object} model.Response{msg=model.Contest} "成功返回赛事详情"
// @Router /contests/{contest_id} [get]
func getContestByID(c *gin.Context) {
	contestIDString := c.Param("contest_id")
	contestID, err := strconv.Atoi(contestIDString)
	if err != nil {
		response.RespError(c, 400, "Invalid contest ID")
		return
	}

	contest, err := getContestByIDFn(contestID)
	if err != nil {
		if errors.Is(err, errContestNotFound) {
			response.RespError(c, 404, "Contest not found")
			return
		}
		log.Logger.Warn("Failed to get contest by id src: " + err.Error())
		response.RespError(c, 500, "Failed to get contest")
		return
	}

	response.RespSuccess(c, contest)
}

// @Summary 获取赛道列表
// @Description 获取指定赛事下的所有赛道列表，包括基本信息如赛道ID、名称、状态等
// @Tags Tracks
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param contest_id path int true "赛事ID"
// @Success 200 {object} model.Response{msg=[]model.Track} "成功返回赛道列表"
// @Router /tracks/{contest_id} [get]
func getTracks(c *gin.Context) {
	contestIDString := c.Param("contest_id")
	contestID, err := strconv.Atoi(contestIDString)
	if err != nil {
		response.RespError(c, 400, "Invalid contest ID")
		return
	}

	tracks, err := getTracksSrcFn(contestID)
	if err != nil {
		log.Logger.Warn("Failed to get tracks src: " + err.Error())
		response.RespError(c, 500, "Failed to get tracks.")
		return
	}

	response.RespSuccess(c, tracks)
}

// @Summary 获取赛道详情
// @Description 获取指定赛道的详细信息
// @Tags Tracks
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param track_id path int true "赛道ID"
// @Success 200 {object} model.Response{msg=model.Track} "成功返回赛道详情"
// @Router /tracks/detail/{track_id} [get]
func getTrackByID(c *gin.Context) {
	trackIDString := c.Param("track_id")
	trackID, err := strconv.Atoi(trackIDString)
	if err != nil {
		response.RespError(c, 400, "Invalid track ID")
		return
	}

	track, err := getTrackByIDSrcFn(trackID)
	if err != nil {
		if errors.Is(err, errTrackNotFound) {
			response.RespError(c, 404, "Track not found")
			return
		}
		log.Logger.Warn("Failed to get track by id src: " + err.Error())
		response.RespError(c, 500, "Failed to get track")
		return
	}

	response.RespSuccess(c, track)
}
