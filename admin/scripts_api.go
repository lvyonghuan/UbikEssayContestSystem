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

var (
	createScriptDefinitionSrcFn     = createScriptDefinitionSrc
	listScriptDefinitionsSrcFn      = listScriptDefinitionsSrc
	getScriptDefinitionByIDSrcFn    = getScriptDefinitionByIDSrc
	updateScriptDefinitionSrcFn     = updateScriptDefinitionSrc
	setScriptDefinitionEnabledSrcFn = setScriptDefinitionEnabledSrc
	uploadScriptVersionSrcFn        = uploadScriptVersionSrc
	listScriptVersionsSrcFn         = listScriptVersionsSrc
	activateScriptVersionSrcFn      = activateScriptVersionSrc

	createScriptFlowSrcFn     = createScriptFlowSrc
	listScriptFlowsSrcFn      = listScriptFlowsSrc
	getScriptFlowByIDSrcFn    = getScriptFlowByIDSrc
	updateScriptFlowSrcFn     = updateScriptFlowSrc
	setScriptFlowEnabledSrcFn = setScriptFlowEnabledSrc
	replaceFlowStepsSrcFn     = replaceFlowStepsSrc
	listFlowStepsSrcFn        = listFlowStepsSrc
	createFlowMountSrcFn      = createFlowMountSrc
	deleteFlowMountSrcFn      = deleteFlowMountSrc
	listFlowMountsByFlowSrcFn = listFlowMountsByFlowSrc
)

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

// @Summary 创建脚本定义
// @Description 管理员创建脚本定义
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param req body model.ScriptDefinition true "脚本定义"
// @Success 200 {object} model.Response{msg=model.ScriptDefinition} "创建成功"
// @Router /admin/scripts [post]
func createScriptDefinition(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	var req model.ScriptDefinition
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err := createScriptDefinitionSrcFn(adminID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, req)
}

// @Summary 获取脚本定义列表
// @Description 管理员获取脚本定义列表
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Success 200 {object} model.Response{msg=[]model.ScriptDefinition} "获取成功"
// @Router /admin/scripts [get]
func listScriptDefinitions(c *gin.Context) {
	defs, err := listScriptDefinitionsSrcFn()
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, defs)
}

// @Summary 获取脚本定义详情
// @Description 管理员根据脚本ID获取脚本定义
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param script_id path int true "脚本ID"
// @Success 200 {object} model.Response{msg=model.ScriptDefinition} "获取成功"
// @Router /admin/scripts/{script_id} [get]
func getScriptDefinitionByID(c *gin.Context) {
	scriptID, err := strconv.Atoi(c.Param("script_id"))
	if err != nil {
		response.RespError(c, 400, "invalid script_id")
		return
	}

	def, err := getScriptDefinitionByIDSrcFn(scriptID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, def)
}

// @Summary 更新脚本定义
// @Description 管理员更新指定脚本定义
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param script_id path int true "脚本ID"
// @Param req body model.ScriptDefinition true "更新后的脚本定义"
// @Success 200 {object} model.Response{msg=model.ScriptDefinition} "更新成功"
// @Router /admin/scripts/{script_id} [put]
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

	if err = updateScriptDefinitionSrcFn(adminID, scriptID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, req)
}

// @Summary 更新脚本启用状态
// @Description 管理员更新指定脚本的启用状态
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param script_id path int true "脚本ID"
// @Param req body updateStatusRequest true "状态请求"
// @Success 200 {object} model.Response "更新成功"
// @Router /admin/scripts/{script_id}/status [post]
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

	if err = setScriptDefinitionEnabledSrcFn(adminID, scriptID, req.IsEnabled); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, nil)
}

// @Summary 上传脚本版本
// @Description 管理员上传脚本文件并创建新版本
// @Tags Admin
// @Accept multipart/form-data
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param script_id path int true "脚本ID"
// @Param script_file formData file true "脚本文件"
// @Success 200 {object} model.Response{msg=model.ScriptVersion} "上传成功"
// @Router /admin/scripts/{script_id}/versions/upload [post]
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

	version, err := uploadScriptVersionSrcFn(adminID, scriptID, fileHeader)
	if err != nil {
		log.Logger.Warn("Upload script version error: " + err.Error())
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, version)
}

// @Summary 获取脚本版本列表
// @Description 管理员获取指定脚本的版本列表
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param script_id path int true "脚本ID"
// @Success 200 {object} model.Response{msg=[]model.ScriptVersion} "获取成功"
// @Router /admin/scripts/{script_id}/versions [get]
func listScriptVersions(c *gin.Context) {
	scriptID, err := strconv.Atoi(c.Param("script_id"))
	if err != nil {
		response.RespError(c, 400, "invalid script_id")
		return
	}

	versions, err := listScriptVersionsSrcFn(scriptID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, versions)
}

// @Summary 激活脚本版本
// @Description 管理员激活指定脚本版本
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param script_id path int true "脚本ID"
// @Param version_id path int true "版本ID"
// @Success 200 {object} model.Response "激活成功"
// @Router /admin/scripts/{script_id}/versions/{version_id}/activate [post]
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

	if err = activateScriptVersionSrcFn(adminID, scriptID, versionID); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, nil)
}

// @Summary 创建脚本流程
// @Description 管理员创建脚本流程
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param req body model.ScriptFlow true "脚本流程"
// @Success 200 {object} model.Response{msg=model.ScriptFlow} "创建成功"
// @Router /admin/script-flows [post]
func createScriptFlow(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	var req model.ScriptFlow
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err := createScriptFlowSrcFn(adminID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, req)
}

// @Summary 获取脚本流程列表
// @Description 管理员获取脚本流程列表
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Success 200 {object} model.Response{msg=[]model.ScriptFlow} "获取成功"
// @Router /admin/script-flows [get]
func listScriptFlows(c *gin.Context) {
	flows, err := listScriptFlowsSrcFn()
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, flows)
}

// @Summary 获取脚本流程详情
// @Description 管理员根据流程ID获取脚本流程
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param flow_id path int true "流程ID"
// @Success 200 {object} model.Response{msg=model.ScriptFlow} "获取成功"
// @Router /admin/script-flows/{flow_id} [get]
func getScriptFlowByID(c *gin.Context) {
	flowID, err := strconv.Atoi(c.Param("flow_id"))
	if err != nil {
		response.RespError(c, 400, "invalid flow_id")
		return
	}

	flow, err := getScriptFlowByIDSrcFn(flowID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, flow)
}

// @Summary 更新脚本流程
// @Description 管理员更新指定脚本流程
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param flow_id path int true "流程ID"
// @Param req body model.ScriptFlow true "更新后的流程"
// @Success 200 {object} model.Response{msg=model.ScriptFlow} "更新成功"
// @Router /admin/script-flows/{flow_id} [put]
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

	if err = updateScriptFlowSrcFn(adminID, flowID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, req)
}

// @Summary 更新流程启用状态
// @Description 管理员更新指定流程的启用状态
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param flow_id path int true "流程ID"
// @Param req body updateStatusRequest true "状态请求"
// @Success 200 {object} model.Response "更新成功"
// @Router /admin/script-flows/{flow_id}/status [post]
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

	if err = setScriptFlowEnabledSrcFn(adminID, flowID, req.IsEnabled); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 替换流程步骤
// @Description 管理员替换指定流程的全部步骤
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param flow_id path int true "流程ID"
// @Param req body []model.FlowStep true "步骤列表"
// @Success 200 {object} model.Response "更新成功"
// @Router /admin/script-flows/{flow_id}/steps [put]
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

	if err = replaceFlowStepsSrcFn(adminID, flowID, req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 获取流程步骤列表
// @Description 管理员获取指定流程步骤
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param flow_id path int true "流程ID"
// @Success 200 {object} model.Response{msg=[]model.FlowStep} "获取成功"
// @Router /admin/script-flows/{flow_id}/steps [get]
func listFlowSteps(c *gin.Context) {
	flowID, err := strconv.Atoi(c.Param("flow_id"))
	if err != nil {
		response.RespError(c, 400, "invalid flow_id")
		return
	}

	steps, err := listFlowStepsSrcFn(flowID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, steps)
}

// @Summary 创建流程挂载
// @Description 管理员创建流程挂载
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param req body model.FlowMount true "挂载信息"
// @Success 200 {object} model.Response{msg=model.FlowMount} "创建成功"
// @Router /admin/script-flows/mounts [post]
func createFlowMount(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	var req model.FlowMount
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err := createFlowMountSrcFn(adminID, &req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, req)
}

// @Summary 删除流程挂载
// @Description 管理员删除指定流程挂载
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param mount_id path int true "挂载ID"
// @Success 200 {object} model.Response "删除成功"
// @Router /admin/script-flows/mounts/{mount_id} [delete]
func deleteFlowMount(c *gin.Context) {
	adminID := c.GetInt("admin_token_id")
	mountID, err := strconv.Atoi(c.Param("mount_id"))
	if err != nil {
		response.RespError(c, 400, "invalid mount_id")
		return
	}

	if err = deleteFlowMountSrcFn(adminID, mountID); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 获取流程挂载列表
// @Description 管理员获取指定流程的挂载列表
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param flow_id path int true "流程ID"
// @Success 200 {object} model.Response{msg=[]model.FlowMount} "获取成功"
// @Router /admin/script-flows/{flow_id}/mounts [get]
func listFlowMounts(c *gin.Context) {
	flowID, err := strconv.Atoi(c.Param("flow_id"))
	if err != nil {
		response.RespError(c, 400, "invalid flow_id")
		return
	}

	mounts, err := listFlowMountsByFlowSrcFn(flowID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, mounts)
}
