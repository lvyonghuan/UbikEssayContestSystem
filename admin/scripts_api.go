package admin

import (
	"main/model"
	"main/util/log"
	"main/util/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type updateStatusRequest struct {
	IsEnabled bool `json:"isEnabled"`
}

func registerScriptRoutes(admin *gin.RouterGroup) {
	scripts := admin.Group("/scripts", checkAccessToken)
	{
		scripts.POST("", createScriptDefinition)
		scripts.GET("", listScriptDefinitions)
		scripts.GET("/:script_id", getScriptDefinitionByID)
		scripts.PUT("/:script_id", updateScriptDefinition)
		scripts.POST("/:script_id/status", updateScriptDefinitionStatus)
		scripts.POST("/:script_id/versions/upload", uploadScriptVersion)
		scripts.GET("/:script_id/versions", listScriptVersions)
		scripts.POST("/:script_id/versions/:version_id/activate", activateScriptVersion)
	}

	flows := admin.Group("/script-flows", checkAccessToken)
	{
		flows.POST("", createScriptFlow)
		flows.GET("", listScriptFlows)
		flows.GET("/:flow_id", getScriptFlowByID)
		flows.PUT("/:flow_id", updateScriptFlow)
		flows.POST("/:flow_id/status", updateScriptFlowStatus)
		flows.PUT("/:flow_id/steps", replaceFlowSteps)
		flows.GET("/:flow_id/steps", listFlowSteps)
		flows.POST("/mounts", createFlowMount)
		flows.DELETE("/mounts/:mount_id", deleteFlowMount)
		flows.GET("/:flow_id/mounts", listFlowMounts)
	}
}

func createScriptDefinition(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	var req model.ScriptDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err := createScriptDefinitionSrc(adminID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, req)
}

func listScriptDefinitions(c *gin.Context) {
	defs, err := listScriptDefinitionsSrc()
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, defs)
}

func getScriptDefinitionByID(c *gin.Context) {
	scriptID, err := strconv.Atoi(c.Param("script_id"))
	if err != nil {
		response.RespError(c, 400, "invalid script_id")
		return
	}

	def, err := getScriptDefinitionByIDSrc(scriptID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, def)
}

func updateScriptDefinition(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	scriptID, err := strconv.Atoi(c.Param("script_id"))
	if err != nil {
		response.RespError(c, 400, "invalid script_id")
		return
	}

	var req model.ScriptDefinition
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err = updateScriptDefinitionSrc(adminID, scriptID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, req)
}

func updateScriptDefinitionStatus(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	scriptID, err := strconv.Atoi(c.Param("script_id"))
	if err != nil {
		response.RespError(c, 400, "invalid script_id")
		return
	}

	var req updateStatusRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err = setScriptDefinitionEnabledSrc(adminID, scriptID, req.IsEnabled); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, nil)
}

func uploadScriptVersion(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	scriptID, err := strconv.Atoi(c.Param("script_id"))
	if err != nil {
		response.RespError(c, 400, "invalid script_id")
		return
	}

	fileHeader, err := c.FormFile("script_file")
	if err != nil {
		response.RespError(c, 400, "script_file is required")
		return
	}

	version, err := uploadScriptVersionSrc(adminID, scriptID, fileHeader)
	if err != nil {
		log.Logger.Warn("Upload script version error: " + err.Error())
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, version)
}

func listScriptVersions(c *gin.Context) {
	scriptID, err := strconv.Atoi(c.Param("script_id"))
	if err != nil {
		response.RespError(c, 400, "invalid script_id")
		return
	}

	versions, err := listScriptVersionsSrc(scriptID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, versions)
}

func activateScriptVersion(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	scriptID, err := strconv.Atoi(c.Param("script_id"))
	if err != nil {
		response.RespError(c, 400, "invalid script_id")
		return
	}
	versionID, err := strconv.Atoi(c.Param("version_id"))
	if err != nil {
		response.RespError(c, 400, "invalid version_id")
		return
	}

	if err = activateScriptVersionSrc(adminID, scriptID, versionID); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, nil)
}

func createScriptFlow(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	var req model.ScriptFlow
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err := createScriptFlowSrc(adminID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, req)
}

func listScriptFlows(c *gin.Context) {
	flows, err := listScriptFlowsSrc()
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, flows)
}

func getScriptFlowByID(c *gin.Context) {
	flowID, err := strconv.Atoi(c.Param("flow_id"))
	if err != nil {
		response.RespError(c, 400, "invalid flow_id")
		return
	}

	flow, err := getScriptFlowByIDSrc(flowID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, flow)
}

func updateScriptFlow(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	flowID, err := strconv.Atoi(c.Param("flow_id"))
	if err != nil {
		response.RespError(c, 400, "invalid flow_id")
		return
	}

	var req model.ScriptFlow
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err = updateScriptFlowSrc(adminID, flowID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, req)
}

func updateScriptFlowStatus(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	flowID, err := strconv.Atoi(c.Param("flow_id"))
	if err != nil {
		response.RespError(c, 400, "invalid flow_id")
		return
	}

	var req updateStatusRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err = setScriptFlowEnabledSrc(adminID, flowID, req.IsEnabled); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

func replaceFlowSteps(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	flowID, err := strconv.Atoi(c.Param("flow_id"))
	if err != nil {
		response.RespError(c, 400, "invalid flow_id")
		return
	}

	var req []model.FlowStep
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err = replaceFlowStepsSrc(adminID, flowID, req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

func listFlowSteps(c *gin.Context) {
	flowID, err := strconv.Atoi(c.Param("flow_id"))
	if err != nil {
		response.RespError(c, 400, "invalid flow_id")
		return
	}

	steps, err := listFlowStepsSrc(flowID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, steps)
}

func createFlowMount(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	var req model.FlowMount
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err := createFlowMountSrc(adminID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, req)
}

func deleteFlowMount(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	mountID, err := strconv.Atoi(c.Param("mount_id"))
	if err != nil {
		response.RespError(c, 400, "invalid mount_id")
		return
	}

	if err = deleteFlowMountSrc(adminID, mountID); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

func listFlowMounts(c *gin.Context) {
	flowID, err := strconv.Atoi(c.Param("flow_id"))
	if err != nil {
		response.RespError(c, 400, "invalid flow_id")
		return
	}

	mounts, err := listFlowMountsByFlowSrc(flowID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, mounts)
}
