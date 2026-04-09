package admin

import (
	"encoding/csv"
	"errors"
	"io"
	"main/model"
	"main/util/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func isBadRequestError(err error) bool {
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if msg == "" {
		return false
	}
	if strings.Contains(msg, "invalid") || strings.Contains(msg, "bad request") || strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
		return true
	}
	return false
}

// @Summary 获取子管理员列表
// @Description 仅超级管理员可获取全部子管理员及其权限
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Success 200 {object} model.Response{msg=[]model.SubAdminInfo} "获取成功"
// @Router /admin/sub-admins [get]
func listSubAdmins(c *gin.Context) {
	subAdmins, err := listSubAdminsSrcFn()
	if err != nil {
		response.RespError(c, 500, "error: list sub admins failed")
		return
	}

	response.RespSuccess(c, subAdmins)
}

// @Summary 创建子管理员
// @Description 仅超级管理员可创建子管理员并分配权限，返回临时密码并尝试发送邮件
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param req body model.CreateSubAdminRequest true "创建子管理员请求"
// @Success 200 {object} model.Response{msg=model.SubAdminCreateResult} "创建成功"
// @Router /admin/sub-admins [post]
func createSubAdmin(c *gin.Context) {
	var req model.CreateSubAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "error: invalid request body")
		return
	}

	result, err := createSubAdminSrcFn(c.GetInt("admin_token_id"), req)
	if err != nil {
		if isBadRequestError(err) {
			response.RespError(c, 400, err.Error())
			return
		}
		response.RespError(c, 500, "error: create sub admin failed")
		return
	}

	response.RespSuccess(c, result)
}

func parseEmailsFromCSV(reader io.Reader) ([]string, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1

	emails := make([]string, 0)
	rowIndex := 0
	for {
		record, err := csvReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) == 0 {
			rowIndex++
			continue
		}

		email := strings.TrimSpace(record[0])
		if email == "" {
			rowIndex++
			continue
		}
		if rowIndex == 0 && !strings.Contains(email, "@") {
			rowIndex++
			continue
		}

		emails = append(emails, email)
		rowIndex++
	}

	return emails, nil
}

func parsePermissionNamesFromQuery(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		result = append(result, p)
	}
	return result
}

// @Summary 批量创建子管理员
// @Description 支持 JSON({emails:[...]}) 或 text/csv(首列邮箱) 的批量导入，返回每条创建结果与失败原因
// @Tags Admin
// @Accept application/json,text/csv
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param req body model.BatchCreateSubAdminsRequest false "JSON批量创建请求"
// @Success 200 {object} model.Response{msg=model.BatchCreateSubAdminsResponse} "创建完成"
// @Router /admin/sub-admins/batch [post]
func batchCreateSubAdmins(c *gin.Context) {
	contentType := strings.ToLower(strings.TrimSpace(c.ContentType()))
	emails := make([]string, 0)
	permissionNames := make([]string, 0)

	if strings.Contains(contentType, "application/json") {
		var req model.BatchCreateSubAdminsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.RespError(c, 400, "error: invalid request body")
			return
		}
		emails = req.Emails
		permissionNames = req.PermissionNames
	} else {
		parsedEmails, err := parseEmailsFromCSV(c.Request.Body)
		if err != nil {
			response.RespError(c, 400, "error: invalid csv body")
			return
		}
		emails = parsedEmails
		permissionNames = parsePermissionNamesFromQuery(c.Query("permission_names"))
	}

	if len(emails) == 0 {
		response.RespError(c, 400, "error: emails is required")
		return
	}

	result, err := batchCreateSubAdminsSrcFn(c.GetInt("admin_token_id"), emails, permissionNames)
	if err != nil {
		if isBadRequestError(err) {
			response.RespError(c, 400, err.Error())
			return
		}
		response.RespError(c, 500, "error: batch create sub admins failed")
		return
	}

	response.RespSuccess(c, result)
}

// @Summary 更新子管理员权限
// @Description 仅超级管理员可替换子管理员的权限集合
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param admin_id path int true "子管理员ID"
// @Param req body model.UpdateSubAdminPermissionsRequest true "权限更新请求"
// @Success 200 {object} model.Response{} "更新成功"
// @Router /admin/sub-admins/{admin_id}/permissions [put]
func updateSubAdminPermissions(c *gin.Context) {
	targetAdminID, err := strconv.Atoi(c.Param("admin_id"))
	if err != nil || targetAdminID <= 0 {
		response.RespError(c, 400, "error: invalid admin_id")
		return
	}

	var req model.UpdateSubAdminPermissionsRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "error: invalid request body")
		return
	}

	err = updateSubAdminPermissionsFn(c.GetInt("admin_token_id"), targetAdminID, req.PermissionNames)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.RespError(c, 404, "error: sub admin not found")
			return
		}
		if isBadRequestError(err) {
			response.RespError(c, 400, err.Error())
			return
		}
		response.RespError(c, 500, "error: update sub admin permissions failed")
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 禁用子管理员
// @Description 仅超级管理员可禁用子管理员账户
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param admin_id path int true "子管理员ID"
// @Success 200 {object} model.Response{} "禁用成功"
// @Router /admin/sub-admins/{admin_id}/disable [post]
func disableSubAdmin(c *gin.Context) {
	targetAdminID, err := strconv.Atoi(c.Param("admin_id"))
	if err != nil || targetAdminID <= 0 {
		response.RespError(c, 400, "error: invalid admin_id")
		return
	}

	err = disableSubAdminSrcFn(c.GetInt("admin_token_id"), targetAdminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.RespError(c, 404, "error: sub admin not found")
			return
		}
		if isBadRequestError(err) {
			response.RespError(c, 400, err.Error())
			return
		}
		response.RespError(c, 500, "error: disable sub admin failed")
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 删除子管理员
// @Description 仅超级管理员可删除子管理员账户
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param admin_id path int true "子管理员ID"
// @Success 200 {object} model.Response{} "删除成功"
// @Router /admin/sub-admins/{admin_id} [delete]
func deleteSubAdmin(c *gin.Context) {
	targetAdminID, err := strconv.Atoi(c.Param("admin_id"))
	if err != nil || targetAdminID <= 0 {
		response.RespError(c, 400, "error: invalid admin_id")
		return
	}

	err = deleteSubAdminSrcFn(c.GetInt("admin_token_id"), targetAdminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.RespError(c, 404, "error: sub admin not found")
			return
		}
		if isBadRequestError(err) {
			response.RespError(c, 400, err.Error())
			return
		}
		response.RespError(c, 500, "error: delete sub admin failed")
		return
	}

	response.RespSuccess(c, nil)
}

// @Summary 超级管理员交接
// @Description 将超级管理员权限交接给目标管理员，并禁用当前超级管理员
// @Tags Admin
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer {access_token}"
// @Param req body model.HandoverSuperAdminRequest true "交接请求"
// @Success 200 {object} model.Response{} "交接成功"
// @Router /admin/sub-admins/handover-super [post]
func handoverSuperAdmin(c *gin.Context) {
	var req model.HandoverSuperAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "error: invalid request body")
		return
	}
	if req.NewSuperAdminID <= 0 {
		response.RespError(c, 400, "error: invalid newSuperAdminID")
		return
	}

	err := handoverSuperAdminSrcFn(c.GetInt("admin_token_id"), req.NewSuperAdminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.RespError(c, 404, "error: target admin not found")
			return
		}
		if isBadRequestError(err) {
			response.RespError(c, 400, err.Error())
			return
		}
		response.RespError(c, 500, "error: handover super admin failed")
		return
	}

	response.RespSuccess(c, nil)
}
