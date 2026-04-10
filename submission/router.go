package submission

import (
	"main/conf"
	_ "main/docs/API/Submission"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/response"
	"main/util/token"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	checkTokenFn = token.CheckToken
	runServerFn  = func(r *gin.Engine, port string) error {
		return r.Run(":" + port)
	}
)

// InitRouter
// @title          Submission API
// @version         1.0
// @description     Ubik 系统提交接口文档
// @host            localhost:80
// @BasePath        /api/v1
func InitRouter(conf conf.APIConfig) {
	r := BuildSubmissionRouter()
	_ = runServerFn(r, conf.SubmissionsPort)
}

func BuildSubmissionRouter() *gin.Engine {
	return buildSubmissionRouter()
}

func buildSubmissionRouter() *gin.Engine {
	r := gin.Default()

	// 鎸傝浇swagger璺敱
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
				submission.GET("", checkAccessToken, getSubmissions)
				submission.POST("/file", checkAccessToken, saveSubmissionFile) //FIXME 上传文档完整性校验？大小限制？或者上传速率限制？
				submission.GET("/file/:submission_id", checkAccessToken, getSubmissionFile)
				submission.POST("", checkAccessToken, checkWorkSubmissionValid, submissionWork)
				submission.PUT("", checkAccessToken, checkWorkSubmissionValid, updateSubmission)
				submission.DELETE("", checkAccessToken, checkWorkSubmissionValid, deleteSubmission)
			}
		}
	}

	return r
}

func checkAccessToken(c *gin.Context) {
	bearerToken := c.GetHeader("Authorization")

	id, role, err := checkTokenFn(bearerToken)
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

	tokenAuthorID := c.GetInt("author_token_id")
	if work.AuthorID != 0 && work.AuthorID != tokenAuthorID {
		response.RespError(c, 403, "forbidden: can only operate your own submissions")
		c.Abort()
		return
	}

	work.AuthorID = tokenAuthorID // 从token中获取作者ID，确保提交的作品与当前登录的作者关联

	c.Set("work", work)
	c.Next()
}
