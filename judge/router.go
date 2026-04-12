package judge

import (
	"main/conf"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter(conf conf.APIConfig) {

}

func BuildJudgeRouter() *gin.Engine {

}

func buildJudgeRouter() *gin.Engine {
	r := gin.Default()

	//挂载swagger文件路由
	r.GET("/swaager/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("Judge")))

	v1 := r.Group("/api/v1")
	{
		//评委没有注册功能，由管理员统一导入生成账号
		judge := v1.Group("/judge")
		{
			judge.POST("/login") //返回当前评委Token
		}

		review := v1.Group("/review")
		{
			review.GET("/events") //返回和当前评委相关的评审事件
			// 注：判断评委关联性可以使用redis作为缓冲，但是如果管理员进行了修改（例如调整了评审事件的评委），需要及时更新redis中的数据
			review.GET("/:event_id") //返回指定的评审事件（只能是和当前评委有关的）
			//返回和指定评审事件关联的作品列表，要求只能是进行中的评审，且只能是和该评委有关的评审事件。
			//获取粒度为：指定赛道ID的指定状态的作品（也就是评审事件表给出的相关字段）
			review.GET("/judge/:event_id")
			review.GET("/juedge/file")           //获取作品PDF，传入赛道-作品-作者ID三段定位
			review.POST("/judge/result")         //上传作品分数、评语等，可在jsonb字段写入维度打分
			review.GET("judge/result/:event_id") //获取已有的作品评分结果，要求只能是和当前评委有关的评审事件
			review.PUT("/judege/:result_id")     //更新作品评分结果
		}
	}

	return r
}
