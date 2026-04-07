package system

import (
	"main/conf"
	"main/util/log"
	"main/util/response"
	"main/util/token"
	"strconv"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// initGlobalInfoRouter 只能有GET
// @title           全局信息 API
// @version         1.0
// @description     Ubik 系统全局信息接口文档
// @host            localhost:8082
// @BasePath        /api/v1
func initGlobalInfoRouter(apiConf conf.APIConfig) {
	r := gin.Default()

	// 挂载swagger路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("GlobalInfo")))

	//这里的所有操作都需要先check一个合法token，避免ddos
	v1 := r.Group("/api/v1", checkAccessToken)
	{
		contests := v1.Group("contests")
		{
			contests.GET("", getContests) //获取赛事列表
			//contests.GET("/:contest_id") //获取赛事详情（包括开始-结束时间等等）
		}

		tracks := v1.Group("tracks")
		{
			tracks.GET("/:contest_id", getTracks) // 获取赛事下赛道列表
			//tracks.GET("/:track_id")   // 获取赛道详情
		}
	}

	r.Run(":" + apiConf.GlobalInfoPort)
}

// --- 中间件
func checkAccessToken(c *gin.Context) {
	bearerToken := c.GetHeader("Authorization")

	if bearerToken == "" {
		response.RespError(c, 401, "Unauthorized: No token provided")
		c.Abort()
		return
	}

	_, _, err := token.CheckToken(bearerToken)
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
// @Success 200 {object} model.Response{msg=[]model.Track} "成功反回赛事列表"
// @Router /contests/ [get]
func getContests(c *gin.Context) {
	contests, err := getContestSrc()
	if err != nil {
		log.Logger.Warn("Failed to get contest src: " + err.Error())
		response.RespError(c, 500, "Failed to get contests")
	}

	response.RespSuccess(c, contests)
}

// @Summary 获取赛道列表
// @Description 获取指定赛事下的所有赛道列表，包括基本信息如赛道ID、名称、状态等
// @Tags Tracks
// @Accept application/json
// @Produce application/json
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

	tracks, err := getTracksSrc(contestID)
	if err != nil {
		log.Logger.Warn("Failed to get tracks src: " + err.Error())
		response.RespError(c, 500, "Failed to get tracks.")
		return
	}

	response.RespSuccess(c, tracks)
}
