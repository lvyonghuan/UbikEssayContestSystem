package judge

import (
	"main/conf"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter(conf conf.APIConfig) {
	r := BuildJudgeRouter()
	_ = runServerFn(r, conf.JudgePort)
}

func BuildJudgeRouter() *gin.Engine {
	return buildJudgeRouter()
}

func buildJudgeRouter() *gin.Engine {
	r := gin.Default()

	//挂载swagger文件路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("Judge")))

	v1 := r.Group("/api/v1")
	{
		//评委没有注册功能，由管理员统一导入生成账号
		judge := v1.Group("/judge")
		{
			judge.POST("/login", judgeLogin) //返回当前评委Token
		}

		review := v1.Group("/review", checkJudgeAccessToken)
		{
			review.GET("/events", getJudgeEvents) //返回和当前评委相关的评审事件
			// 注：判断评委关联性可以使用redis作为缓冲，但是如果管理员进行了修改（例如调整了评审事件的评委），需要及时更新redis中的数据
			review.GET("/:event_id", getReviewEvent) //返回指定的评审事件（只能是和当前评委有关的）
			//返回和指定评审事件关联的作品列表，要求只能是进行中的评审，且只能是和该评委有关的评审事件。
			//获取粒度为：指定赛道ID的指定状态的作品（也就是评审事件表给出的相关字段）
			review.GET("/judge/:event_id", getEventWorks)
			review.GET("/judge/file", getReviewWorkFile)        //获取作品PDF，传入event_id和work_id
			review.POST("/judge/result", submitReviewResult)    //上传作品分数、评语等，可在jsonb字段写入维度打分
			review.GET("/judge/result/:event_id", getReviewResultsByEvent)
			review.PUT("/judge/:result_id", updateReviewResult)
		}
	}

	return r
}
