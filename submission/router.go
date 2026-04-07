package submission

import (
	"main/conf"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/response"
	"main/util/token"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// InitRouter
// @title           作者端API
// @version         1.0
// @description     Ubik 稿件提交系统接口文档
// @host            localhost:80
// @BasePath        /api/v1
func InitRouter(conf conf.APIConfig) {
	r := gin.Default()

	// 挂载swagger路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("Submission")))
	v1 := r.Group("/api/v1")
	{
		author := v1.Group("/author")
		{
			author.POST("/register", authorRegister)
			author.POST("/login", authorLogin)
			author.GET("/refresh", refreshToken)
			author.PUT("", checkAccessToken, updateAuthor)

			submission := author.Group("/submission")
			{
				submission.GET("/:id", checkAccessToken, getSubmissions)
				submission.POST("/file", checkAccessToken)
				submission.POST("", checkAccessToken, checkWorkSubmissionValid, submissionWork)
				submission.PUT("", checkAccessToken, checkWorkSubmissionValid, updateSubmission)
				submission.DELETE("", checkAccessToken, checkWorkSubmissionValid, deleteSubmission)
			}
		}
	}

	//FIXME 文件操作

	r.Run(":" + conf.SubmissionsPort)
}

func checkAccessToken(c *gin.Context) {
	bearerToken := c.GetHeader("Authorization")

	id, role, err := token.CheckToken(bearerToken)
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

	c.Set("author_token_id", int(id))
	c.Next()
}

func checkWorkSubmissionValid(c *gin.Context) {
	var work model.Work
	err := c.ShouldBind(&work)
	if err != nil {
		log.Logger.Warn("Failed to bind work data: " + err.Error())
		response.RespError(c, 400, "bad request")
		c.Abort()
		return
	}

	if work.AuthorID != c.GetInt("author_token_id") {
		response.RespError(c, 403, "Forbidden")
		c.Abort()
		return
	}

	c.Set("work", work)
	c.Next()
}
