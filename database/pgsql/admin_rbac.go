package pgsql

import (
	"errors"
	"fmt"
	"main/model"
	"sort"
	"strings"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

type dbRole struct {
	RoleID      int    `gorm:"column:role_id"`
	RoleName    string `gorm:"column:role_name"`
	Description string `gorm:"column:description"`
	IsDefault   bool   `gorm:"column:is_default"`
	IsSuper     bool   `gorm:"column:is_super"`
}

type dbPermission struct {
	PermissionID int    `gorm:"column:permission_id"`
	Name         string `gorm:"column:name"`
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
	sort.Strings(result)
	return result
}

func IsAdminActive(adminID int) (bool, error) {
	var admin model.Admin
	result := postgresDB.Select("admin_id", "is_active").Where("admin_id = ?", adminID).First(&admin)
	if result.Error != nil {
		return false, uerr.NewError(result.Error)
	}
	return admin.IsActive, nil
}

func GetAdminByID(adminID int) (model.Admin, error) {
	var admin model.Admin
	result := postgresDB.Where("admin_id = ?", adminID).First(&admin)
	if result.Error != nil {
		return model.Admin{}, uerr.NewError(result.Error)
	}
	return admin, nil
}

func IsAdminSuper(adminID int) (bool, error) {
	var count int64
	result := postgresDB.Table("admin_roles AS ar").
		Joins("JOIN roles AS r ON r.role_id = ar.role_id").
		Where("ar.admin_id = ? AND r.is_super = ?", adminID, true).
		Count(&count)
	if result.Error != nil {
		return false, uerr.NewError(result.Error)
	}
	return count > 0, nil
}

func AdminHasPermission(adminID int, permissionName string) (bool, error) {
	perm := strings.TrimSpace(permissionName)
	if perm == "" {
		return false, nil
	}

	var count int64
	result := postgresDB.Table("admins AS a").
		Joins("JOIN admin_roles AS ar ON ar.admin_id = a.admin_id").
		Joins("JOIN roles AS r ON r.role_id = ar.role_id").
		Joins("LEFT JOIN role_permissions AS rp ON rp.role_id = r.role_id").
		Joins("LEFT JOIN permissions AS p ON p.permission_id = rp.permission_id").
		Where("a.admin_id = ? AND a.is_active = ?", adminID, true).
		Where("(r.is_super = ? OR p.name = ?)", true, perm).
		Count(&count)
	if result.Error != nil {
		return false, uerr.NewError(result.Error)
	}

	return count > 0, nil
}

func ListPermissionNames() ([]string, error) {
	var rows []dbPermission
	result := postgresDB.Table("permissions").Select("permission_id", "name").Order("name ASC").Find(&rows)
	if result.Error != nil {
		return nil, uerr.NewError(result.Error)
	}

	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	return names, nil
}

func ListAdminPermissionNames(adminID int) ([]string, error) {
	var rows []dbPermission
	result := postgresDB.Table("admin_roles AS ar").
		Joins("JOIN roles AS r ON r.role_id = ar.role_id").
		Joins("JOIN role_permissions AS rp ON rp.role_id = r.role_id").
		Joins("JOIN permissions AS p ON p.permission_id = rp.permission_id").
		Select("DISTINCT p.permission_id", "p.name").
		Where("ar.admin_id = ? AND r.is_super = ?", adminID, false).
		Order("p.name ASC").
		Find(&rows)
	if result.Error != nil {
		return nil, uerr.NewError(result.Error)
	}

	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	return names, nil
}

func ensureManagedRoleTx(tx *gorm.DB, roleName, description string) (int, error) {
	var role dbRole
	result := tx.Table("roles").Where("role_name = ?", roleName).First(&role)
	if result.Error == nil {
		return role.RoleID, nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return 0, uerr.NewError(result.Error)
	}

	role = dbRole{
		RoleName:    roleName,
		Description: description,
		IsDefault:   false,
		IsSuper:     false,
	}
	if err := tx.Table("roles").Create(&role).Error; err != nil {
		return 0, uerr.NewError(err)
	}
	return role.RoleID, nil
}

func replaceAdminNonSuperRolesTx(tx *gorm.DB, adminID int, keepRoleID int) error {
	if err := tx.Exec(`
		DELETE FROM admin_roles
		WHERE admin_id = ?
		AND role_id IN (
			SELECT r.role_id FROM roles r WHERE r.is_super = FALSE
		)
	`, adminID).Error; err != nil {
		return uerr.NewError(err)
	}

	mapping := map[string]interface{}{
		"admin_id": adminID,
		"role_id":  keepRoleID,
	}
	if err := tx.Table("admin_roles").Create(mapping).Error; err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func loadPermissionIDsTx(tx *gorm.DB, permissionNames []string) ([]int, error) {
	normalized := normalizePermissionNames(permissionNames)
	if len(normalized) == 0 {
		return nil, nil
	}

	var rows []dbPermission
	if err := tx.Table("permissions").
		Select("permission_id", "name").
		Where("name IN ?", normalized).
		Find(&rows).Error; err != nil {
		return nil, uerr.NewError(err)
	}
	if len(rows) != len(normalized) {
		return nil, uerr.NewError(errors.New("invalid permission names"))
	}

	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.PermissionID)
	}
	return ids, nil
}

func replaceRolePermissionsTx(tx *gorm.DB, roleID int, permissionNames []string) error {
	if err := tx.Table("role_permissions").Where("role_id = ?", roleID).Delete(nil).Error; err != nil {
		return uerr.NewError(err)
	}

	permIDs, err := loadPermissionIDsTx(tx, permissionNames)
	if err != nil {
		return err
	}

	for _, permID := range permIDs {
		mapping := map[string]interface{}{
			"role_id":       roleID,
			"permission_id": permID,
		}
		if createErr := tx.Table("role_permissions").Create(mapping).Error; createErr != nil {
			return uerr.NewError(createErr)
		}
	}

	return nil
}

func CreateSubAdmin(admin *model.Admin, permissionNames []string) error {
	return postgresDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("admins").Create(admin).Error; err != nil {
			return uerr.NewError(err)
		}

		managedRoleName := fmt.Sprintf("subadmin:%d", admin.AdminID)
		roleID, err := ensureManagedRoleTx(tx, managedRoleName, "auto generated role for sub admin")
		if err != nil {
			return err
		}

		if err = replaceAdminNonSuperRolesTx(tx, admin.AdminID, roleID); err != nil {
			return err
		}

		if err = replaceRolePermissionsTx(tx, roleID, permissionNames); err != nil {
			return err
		}

		return nil
	})
}

func SetSubAdminPermissions(adminID int, permissionNames []string) error {
	return postgresDB.Transaction(func(tx *gorm.DB) error {
		var admin model.Admin
		if err := tx.Table("admins").Where("admin_id = ?", adminID).First(&admin).Error; err != nil {
			return uerr.NewError(err)
		}

		isSuper, err := isAdminSuperTx(tx, adminID)
		if err != nil {
			return err
		}
		if isSuper {
			return uerr.NewError(errors.New("cannot update super admin permissions"))
		}

		managedRoleName := fmt.Sprintf("subadmin:%d", adminID)
		roleID, err := ensureManagedRoleTx(tx, managedRoleName, "auto generated role for sub admin")
		if err != nil {
			return err
		}

		if err = replaceAdminNonSuperRolesTx(tx, adminID, roleID); err != nil {
			return err
		}

		if err = replaceRolePermissionsTx(tx, roleID, permissionNames); err != nil {
			return err
		}

		return nil
	})
}

func isAdminSuperTx(tx *gorm.DB, adminID int) (bool, error) {
	var count int64
	if err := tx.Table("admin_roles AS ar").
		Joins("JOIN roles AS r ON r.role_id = ar.role_id").
		Where("ar.admin_id = ? AND r.is_super = ?", adminID, true).
		Count(&count).Error; err != nil {
		return false, uerr.NewError(err)
	}
	return count > 0, nil
}

func ListSubAdmins() ([]model.SubAdminInfo, error) {
	var admins []model.Admin
	result := postgresDB.Table("admins AS a").
		Select("a.admin_id", "a.admin_name", "a.admin_email", "a.is_active").
		Where(`NOT EXISTS (
			SELECT 1 FROM admin_roles AS ar
			JOIN roles AS r ON r.role_id = ar.role_id
			WHERE ar.admin_id = a.admin_id AND r.is_super = TRUE
		)`).
		Order("a.admin_id ASC").
		Find(&admins)
	if result.Error != nil {
		return nil, uerr.NewError(result.Error)
	}

	items := make([]model.SubAdminInfo, 0, len(admins))
	for _, admin := range admins {
		perms, err := ListAdminPermissionNames(admin.AdminID)
		if err != nil {
			return nil, err
		}
		items = append(items, model.SubAdminInfo{
			AdminID:         admin.AdminID,
			AdminName:       admin.AdminName,
			AdminEmail:      admin.AdminEmail,
			IsActive:        admin.IsActive,
			PermissionNames: perms,
		})
	}
	return items, nil
}

func DeleteSubAdminByID(adminID int) error {
	isSuper, err := IsAdminSuper(adminID)
	if err != nil {
		return err
	}
	if isSuper {
		return uerr.NewError(errors.New("cannot delete super admin"))
	}

	result := postgresDB.Table("admins").Where("admin_id = ?", adminID).Delete(&model.Admin{})
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}
	if result.RowsAffected == 0 {
		return uerr.NewError(gorm.ErrRecordNotFound)
	}
	return nil
}

func SetAdminActive(adminID int, isActive bool) error {
	result := postgresDB.Table("admins").Where("admin_id = ?", adminID).Update("is_active", isActive)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}
	if result.RowsAffected == 0 {
		return uerr.NewError(gorm.ErrRecordNotFound)
	}
	return nil
}

func GrantSuperRole(adminID int) error {
	return postgresDB.Transaction(func(tx *gorm.DB) error {
		var admin model.Admin
		if err := tx.Table("admins").Where("admin_id = ?", adminID).First(&admin).Error; err != nil {
			return uerr.NewError(err)
		}

		var role dbRole
		if err := tx.Table("roles").Where("is_super = ?", true).First(&role).Error; err != nil {
			return uerr.NewError(err)
		}

		var count int64
		if err := tx.Table("admin_roles").Where("admin_id = ? AND role_id = ?", adminID, role.RoleID).Count(&count).Error; err != nil {
			return uerr.NewError(err)
		}
		if count == 0 {
			mapping := map[string]interface{}{"admin_id": adminID, "role_id": role.RoleID}
			if err := tx.Table("admin_roles").Create(mapping).Error; err != nil {
				return uerr.NewError(err)
			}
		}

		if err := tx.Table("admins").Where("admin_id = ?", adminID).Update("is_active", true).Error; err != nil {
			return uerr.NewError(err)
		}

		return nil
	})
}

func HandoverSuperAdmin(currentAdminID int, newAdminID int) error {
	if currentAdminID == newAdminID {
		return uerr.NewError(errors.New("new super admin must be different from current"))
	}

	return postgresDB.Transaction(func(tx *gorm.DB) error {
		isCurrentSuper, err := isAdminSuperTx(tx, currentAdminID)
		if err != nil {
			return err
		}
		if !isCurrentSuper {
			return uerr.NewError(errors.New("current admin is not super admin"))
		}

		var newAdmin model.Admin
		if err = tx.Table("admins").Where("admin_id = ?", newAdminID).First(&newAdmin).Error; err != nil {
			return uerr.NewError(err)
		}

		var superRole dbRole
		if err = tx.Table("roles").Where("is_super = ?", true).First(&superRole).Error; err != nil {
			return uerr.NewError(err)
		}

		var mappingCount int64
		if err = tx.Table("admin_roles").Where("admin_id = ? AND role_id = ?", newAdminID, superRole.RoleID).Count(&mappingCount).Error; err != nil {
			return uerr.NewError(err)
		}
		if mappingCount == 0 {
			if err = tx.Table("admin_roles").Create(map[string]interface{}{"admin_id": newAdminID, "role_id": superRole.RoleID}).Error; err != nil {
				return uerr.NewError(err)
			}
		}

		if err = tx.Table("admins").Where("admin_id = ?", newAdminID).Update("is_active", true).Error; err != nil {
			return uerr.NewError(err)
		}
		if err = tx.Table("admins").Where("admin_id = ?", currentAdminID).Update("is_active", false).Error; err != nil {
			return uerr.NewError(err)
		}

		return nil
	})
}

func GetSystemEmailConfig() (model.GlobalConfig, error) {
	return getGlobalConfig()
}
