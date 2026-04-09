package admin

import (
	"errors"
	"main/model"
	"testing"

	"gorm.io/gorm"
)

func TestRBACCheckSrcFunctions(t *testing.T) {
	backupSrcHooks(t)

	isAdminActiveFn = func(adminID int) (bool, error) { return true, nil }
	hasAdminPermissionFn = func(adminID int, permissionName string) (bool, error) {
		if permissionName != "works.read" {
			t.Fatalf("unexpected permission name: %s", permissionName)
		}
		return true, nil
	}
	isAdminSuperFn = func(adminID int) (bool, error) { return adminID == 1, nil }

	active, err := checkAdminActiveSrc(1)
	if err != nil || !active {
		t.Fatalf("checkAdminActiveSrc failed: %v active=%v", err, active)
	}

	hasPerm, err := hasPermissionSrc(1, "works.read")
	if err != nil || !hasPerm {
		t.Fatalf("hasPermissionSrc failed: %v has=%v", err, hasPerm)
	}

	isSuper, err := isSuperAdminSrc(1)
	if err != nil || !isSuper {
		t.Fatalf("isSuperAdminSrc failed: %v super=%v", err, isSuper)
	}
}

func TestCreateAndBatchSubAdminSrc(t *testing.T) {
	backupSrcHooks(t)

	nextID := 100
	listPermissionNamesFn = func() ([]string, error) {
		return []string{"works.read", "works.delete"}, nil
	}
	createSubAdminFn = func(admin *model.Admin, permissionNames []string) error {
		if admin.AdminEmail == "dup@example.com" {
			return errors.New("duplicate key value violates unique constraint")
		}
		nextID++
		admin.AdminID = nextID
		return nil
	}
	getSystemEmailConfigFn = func() (model.GlobalConfig, error) {
		return model.GlobalConfig{EmailAddress: "noreply@example.com", EmailAppPassword: "pwd", EmailSmtpServer: "smtp.example.com", EmailSmtpPort: 587}, nil
	}
	sendSMTPMailFn = func(from string, appPassword string, host string, port int, to string, subject string, body string) error {
		if to == "mailfail@example.com" {
			return errors.New("smtp unreachable")
		}
		return nil
	}
	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {}

	single, err := createSubAdminSrc(1, model.CreateSubAdminRequest{AdminEmail: "ok@example.com", PermissionNames: []string{"works.read"}})
	if err != nil {
		t.Fatalf("createSubAdminSrc failed: %v", err)
	}
	if single.AdminID == 0 || single.TempPassword == "" || !single.EmailSent {
		t.Fatalf("unexpected single create result: %+v", single)
	}

	single, err = createSubAdminSrc(1, model.CreateSubAdminRequest{AdminEmail: "mailfail@example.com", PermissionNames: []string{"works.read"}})
	if err != nil {
		t.Fatalf("createSubAdminSrc should still succeed when email fails: %v", err)
	}
	if single.EmailSent {
		t.Fatalf("expected email failure flag, got %+v", single)
	}

	batch, err := batchCreateSubAdminsSrc(1, []string{"bad", "dup@example.com", "ok2@example.com", "ok2@example.com"}, []string{"works.read"})
	if err != nil {
		t.Fatalf("batchCreateSubAdminsSrc failed: %v", err)
	}
	if len(batch.Created) != 1 {
		t.Fatalf("expected 1 created in batch, got %d", len(batch.Created))
	}
	if len(batch.Failed) != 2 {
		t.Fatalf("expected 2 failed in batch, got %d", len(batch.Failed))
	}
}

func TestSubAdminMutationSrc(t *testing.T) {
	backupSrcHooks(t)

	listPermissionNamesFn = func() ([]string, error) {
		return []string{"works.read", "works.delete"}, nil
	}
	setSubAdminPermsFn = func(adminID int, permissionNames []string) error { return nil }
	setAdminActiveFn = func(adminID int, isActive bool) error { return nil }
	isAdminSuperFn = func(adminID int) (bool, error) { return false, nil }
	deleteSubAdminByIDFn = func(adminID int) error { return nil }
	handoverSuperAdminFn = func(currentAdminID int, newAdminID int) error { return nil }
	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {}

	if err := updateSubAdminPermissionsSrc(1, 9, []string{"works.read"}); err != nil {
		t.Fatalf("updateSubAdminPermissionsSrc failed: %v", err)
	}
	if err := disableSubAdminSrc(1, 9); err != nil {
		t.Fatalf("disableSubAdminSrc failed: %v", err)
	}
	if err := deleteSubAdminSrc(1, 9); err != nil {
		t.Fatalf("deleteSubAdminSrc failed: %v", err)
	}
	if err := handoverSuperAdminSrc(1, 9); err != nil {
		t.Fatalf("handoverSuperAdminSrc failed: %v", err)
	}

	setSubAdminPermsFn = func(adminID int, permissionNames []string) error { return gorm.ErrRecordNotFound }
	if err := updateSubAdminPermissionsSrc(1, 99, []string{"works.read"}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}
