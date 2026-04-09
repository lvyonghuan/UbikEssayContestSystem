package admin

import (
	"errors"
	"fmt"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/password"
	"net"
	netmail "net/mail"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

var (
	isAdminActiveFn        = pgsql.IsAdminActive
	isAdminSuperFn         = pgsql.IsAdminSuper
	hasAdminPermissionFn   = pgsql.AdminHasPermission
	listPermissionNamesFn  = pgsql.ListPermissionNames
	listSubAdminsFn        = pgsql.ListSubAdmins
	createSubAdminFn       = pgsql.CreateSubAdmin
	setSubAdminPermsFn     = pgsql.SetSubAdminPermissions
	deleteSubAdminByIDFn   = pgsql.DeleteSubAdminByID
	setAdminActiveFn       = pgsql.SetAdminActive
	handoverSuperAdminFn   = pgsql.HandoverSuperAdmin
	getSystemEmailConfigFn = pgsql.GetSystemEmailConfig
	sendSMTPMailFn         = sendSMTPMail
)

var subAdminNameCleaner = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func checkAdminActiveSrc(adminID int) (bool, error) {
	active, err := isAdminActiveFn(adminID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("Check admin active error: " + parsedErr.Error())
		return false, parsedErr
	}
	return active, nil
}

func hasPermissionSrc(adminID int, permissionName string) (bool, error) {
	hasPermission, err := hasAdminPermissionFn(adminID, permissionName)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("Check admin permission error: " + parsedErr.Error())
		return false, parsedErr
	}
	return hasPermission, nil
}

func isSuperAdminSrc(adminID int) (bool, error) {
	isSuper, err := isAdminSuperFn(adminID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("Check super admin error: " + parsedErr.Error())
		return false, parsedErr
	}
	return isSuper, nil
}

func normalizePermissionNames(permissionNames []string) []string {
	seen := make(map[string]struct{}, len(permissionNames))
	result := make([]string, 0, len(permissionNames))
	for _, name := range permissionNames {
		n := strings.TrimSpace(name)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		result = append(result, n)
	}
	return result
}

func validatePermissionNames(permissionNames []string) ([]string, error) {
	normalized := normalizePermissionNames(permissionNames)
	if len(normalized) == 0 {
		return normalized, nil
	}

	available, err := listPermissionNamesFn()
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("List permission names error: " + parsedErr.Error())
		return nil, parsedErr
	}

	availableSet := make(map[string]struct{}, len(available))
	for _, name := range available {
		availableSet[name] = struct{}{}
	}

	for _, name := range normalized {
		if _, ok := availableSet[name]; !ok {
			return nil, errors.New("invalid permission name: " + name)
		}
	}

	return normalized, nil
}

func generateSubAdminNameByEmail(email string) string {
	local := strings.TrimSpace(strings.Split(email, "@")[0])
	local = strings.ToLower(local)
	local = subAdminNameCleaner.ReplaceAllString(local, "_")
	local = strings.Trim(local, "_")
	if local == "" {
		local = "subadmin"
	}
	if len(local) > 24 {
		local = local[:24]
	}

	return fmt.Sprintf("%s_%d", local, time.Now().UnixNano()%1000000)
}

func sendSMTPMail(from string, appPassword string, host string, port int, to string, subject string, body string) error {
	from = strings.TrimSpace(from)
	host = strings.TrimSpace(host)
	to = strings.TrimSpace(to)
	if from == "" || appPassword == "" || host == "" || port <= 0 {
		return errors.New("email config is incomplete")
	}
	if to == "" {
		return errors.New("email receiver is empty")
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	auth := smtp.PlainAuth("", from, appPassword, host)
	message := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(message))
}

func sendSubAdminPasswordEmail(email string, adminName string, tempPassword string) error {
	emailCfg, err := getSystemEmailConfigFn()
	if err != nil {
		return uerr.ExtractError(err)
	}

	subject := "Ubik 子管理员账户创建通知"
	body := fmt.Sprintf("你好，%s：\n\n你的 Ubik 管理员账户已创建。\n用户名: %s\n临时密码: %s\n\n请登录后尽快修改密码。", adminName, adminName, tempPassword)

	return sendSMTPMailFn(
		emailCfg.EmailAddress,
		emailCfg.EmailAppPassword,
		emailCfg.EmailSmtpServer,
		emailCfg.EmailSmtpPort,
		email,
		subject,
		body,
	)
}

func createSubAdminWithValidatedPermissions(operatorAdminID int, adminName string, adminEmail string, permissionNames []string) (model.SubAdminCreateResult, error) {
	tempPassword := password.Generate()
	hashedPassword, err := password.HashPassword(tempPassword)
	if err != nil {
		return model.SubAdminCreateResult{}, uerr.ExtractError(err)
	}

	if strings.TrimSpace(adminName) == "" {
		adminName = generateSubAdminNameByEmail(adminEmail)
	}

	admin := &model.Admin{
		AdminName:  strings.TrimSpace(adminName),
		Password:   hashedPassword,
		AdminEmail: adminEmail,
		IsActive:   true,
	}

	if err = createSubAdminFn(admin, permissionNames); err != nil {
		return model.SubAdminCreateResult{}, uerr.ExtractError(err)
	}

	result := model.SubAdminCreateResult{
		AdminID:      admin.AdminID,
		AdminName:    admin.AdminName,
		AdminEmail:   admin.AdminEmail,
		TempPassword: tempPassword,
		EmailSent:    true,
	}

	if emailErr := sendSubAdminPasswordEmail(admin.AdminEmail, admin.AdminName, tempPassword); emailErr != nil {
		result.EmailSent = false
		result.EmailError = emailErr.Error()
	}

	createActionLogFn(operatorAdminID, _const.Admins, _const.Create,
		genDetails(
			[]string{"target_admin_id", "target_admin_name", "target_admin_email"},
			[]string{strconv.Itoa(admin.AdminID), admin.AdminName, admin.AdminEmail},
		),
	)

	return result, nil
}

func createSubAdminSrc(operatorAdminID int, req model.CreateSubAdminRequest) (model.SubAdminCreateResult, error) {
	email := strings.ToLower(strings.TrimSpace(req.AdminEmail))
	if _, err := netmail.ParseAddress(email); err != nil {
		return model.SubAdminCreateResult{}, errors.New("invalid adminEmail")
	}

	permissionNames, err := validatePermissionNames(req.PermissionNames)
	if err != nil {
		return model.SubAdminCreateResult{}, err
	}

	return createSubAdminWithValidatedPermissions(operatorAdminID, req.AdminName, email, permissionNames)
}

func batchCreateSubAdminsSrc(operatorAdminID int, emails []string, permissionNames []string) (model.BatchCreateSubAdminsResponse, error) {
	resp := model.BatchCreateSubAdminsResponse{
		Created: make([]model.SubAdminCreateResult, 0),
		Failed:  make([]model.BatchCreateSubAdminFailure, 0),
	}

	validatedPermissions, err := validatePermissionNames(permissionNames)
	if err != nil {
		return resp, err
	}

	seen := make(map[string]struct{}, len(emails))
	for _, raw := range emails {
		email := strings.ToLower(strings.TrimSpace(raw))
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}

		if _, parseErr := netmail.ParseAddress(email); parseErr != nil {
			resp.Failed = append(resp.Failed, model.BatchCreateSubAdminFailure{Email: email, Reason: "invalid email format"})
			continue
		}

		created, createErr := createSubAdminWithValidatedPermissions(operatorAdminID, "", email, validatedPermissions)
		if createErr != nil {
			resp.Failed = append(resp.Failed, model.BatchCreateSubAdminFailure{Email: email, Reason: createErr.Error()})
			continue
		}
		resp.Created = append(resp.Created, created)
	}

	return resp, nil
}

func listSubAdminsSrc() ([]model.SubAdminInfo, error) {
	subAdmins, err := listSubAdminsFn()
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("List sub admins error: " + parsedErr.Error())
		return nil, parsedErr
	}
	return subAdmins, nil
}

func updateSubAdminPermissionsSrc(operatorAdminID int, targetAdminID int, permissionNames []string) error {
	validatedPermissions, err := validatePermissionNames(permissionNames)
	if err != nil {
		return err
	}

	if err = setSubAdminPermsFn(targetAdminID, validatedPermissions); err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		log.Logger.Warn("Update sub admin permissions error: " + parsedErr.Error())
		return parsedErr
	}

	createActionLogFn(operatorAdminID, _const.Admins, _const.Update,
		genDetails(
			[]string{"target_admin_id"},
			[]string{strconv.Itoa(targetAdminID)},
		),
	)

	return nil
}

func disableSubAdminSrc(operatorAdminID int, targetAdminID int) error {
	isSuper, err := isSuperAdminSrc(targetAdminID)
	if err != nil {
		return err
	}
	if isSuper {
		return errors.New("cannot disable super admin")
	}

	if err = setAdminActiveFn(targetAdminID, false); err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		log.Logger.Warn("Disable sub admin error: " + parsedErr.Error())
		return parsedErr
	}

	createActionLogFn(operatorAdminID, _const.Admins, _const.Update,
		genDetails(
			[]string{"target_admin_id", "is_active"},
			[]string{strconv.Itoa(targetAdminID), "false"},
		),
	)
	return nil
}

func deleteSubAdminSrc(operatorAdminID int, targetAdminID int) error {
	if err := deleteSubAdminByIDFn(targetAdminID); err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		log.Logger.Warn("Delete sub admin error: " + parsedErr.Error())
		return parsedErr
	}

	createActionLogFn(operatorAdminID, _const.Admins, _const.Delete,
		genDetails(
			[]string{"target_admin_id"},
			[]string{strconv.Itoa(targetAdminID)},
		),
	)
	return nil
}

func handoverSuperAdminSrc(currentAdminID int, newSuperAdminID int) error {
	if err := handoverSuperAdminFn(currentAdminID, newSuperAdminID); err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		log.Logger.Warn("Handover super admin error: " + parsedErr.Error())
		return parsedErr
	}

	createActionLogFn(currentAdminID, _const.Admins, _const.Update,
		genDetails(
			[]string{"new_super_admin_id", "old_super_admin_disabled"},
			[]string{strconv.Itoa(newSuperAdminID), "true"},
		),
	)

	return nil
}
