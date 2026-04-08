package pgsql

import (
	"errors"
	"main/model"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

var ErrFlowNotMounted = errors.New("script flow not mounted")

type ResolvedFlowStep struct {
	Step    model.FlowStep         `json:"step"`
	Script  model.ScriptDefinition `json:"script"`
	Version model.ScriptVersion    `json:"version"`
}

func CreateScriptDefinition(def *model.ScriptDefinition) error {
	err := postgresDB.Create(def).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func UpdateScriptDefinition(scriptID int, updated *model.ScriptDefinition) error {
	result := postgresDB.Model(&model.ScriptDefinition{}).Where("script_id = ?", scriptID).Updates(updated)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func GetScriptDefinitionByID(scriptID int) (model.ScriptDefinition, error) {
	var def model.ScriptDefinition
	err := postgresDB.Where("script_id = ?", scriptID).First(&def).Error
	if err != nil {
		return model.ScriptDefinition{}, uerr.NewError(err)
	}

	return def, nil
}

func GetScriptDefinitionByKey(scriptKey string) (model.ScriptDefinition, error) {
	var def model.ScriptDefinition
	err := postgresDB.Where("script_key = ?", scriptKey).First(&def).Error
	if err != nil {
		return model.ScriptDefinition{}, uerr.NewError(err)
	}

	return def, nil
}

func ListScriptDefinitions() ([]model.ScriptDefinition, error) {
	var defs []model.ScriptDefinition
	err := postgresDB.Order("script_id ASC").Find(&defs).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return defs, nil
}

func SetScriptDefinitionEnabled(scriptID int, enabled bool) error {
	result := postgresDB.Model(&model.ScriptDefinition{}).Where("script_id = ?", scriptID).Update("is_enabled", enabled)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func GetNextScriptVersionNumber(scriptID int) (int, error) {
	var nextVersion int
	err := postgresDB.Model(&model.ScriptVersion{}).
		Where("script_id = ?", scriptID).
		Select("COALESCE(MAX(version_num), 0) + 1").
		Scan(&nextVersion).Error
	if err != nil {
		return 0, uerr.NewError(err)
	}

	if nextVersion <= 0 {
		nextVersion = 1
	}

	return nextVersion, nil
}

func CreateScriptVersion(version *model.ScriptVersion) error {
	err := postgresDB.Create(version).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func GetScriptVersionByID(versionID int) (model.ScriptVersion, error) {
	var version model.ScriptVersion
	err := postgresDB.Where("version_id = ?", versionID).First(&version).Error
	if err != nil {
		return model.ScriptVersion{}, uerr.NewError(err)
	}

	return version, nil
}

func ListScriptVersions(scriptID int) ([]model.ScriptVersion, error) {
	var versions []model.ScriptVersion
	err := postgresDB.Where("script_id = ?", scriptID).Order("version_num DESC").Find(&versions).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return versions, nil
}

func ActivateScriptVersion(scriptID int, versionID int) error {
	tx := postgresDB.Begin()
	if tx.Error != nil {
		return uerr.NewError(tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var version model.ScriptVersion
	err := tx.Where("version_id = ? AND script_id = ?", versionID, scriptID).First(&version).Error
	if err != nil {
		tx.Rollback()
		return uerr.NewError(err)
	}

	err = tx.Model(&model.ScriptVersion{}).Where("script_id = ?", scriptID).Update("is_active", false).Error
	if err != nil {
		tx.Rollback()
		return uerr.NewError(err)
	}

	err = tx.Model(&model.ScriptVersion{}).
		Where("version_id = ? AND script_id = ?", versionID, scriptID).
		Update("is_active", true).Error
	if err != nil {
		tx.Rollback()
		return uerr.NewError(err)
	}

	err = tx.Commit().Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func GetActiveScriptVersion(scriptID int) (model.ScriptVersion, error) {
	var version model.ScriptVersion
	err := postgresDB.Where("script_id = ? AND is_active = ?", scriptID, true).
		Order("version_num DESC").
		First(&version).Error
	if err != nil {
		return model.ScriptVersion{}, uerr.NewError(err)
	}

	return version, nil
}

func CreateScriptFlow(flow *model.ScriptFlow) error {
	err := postgresDB.Create(flow).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func UpdateScriptFlow(flowID int, updated *model.ScriptFlow) error {
	result := postgresDB.Model(&model.ScriptFlow{}).Where("flow_id = ?", flowID).Updates(updated)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func GetScriptFlowByID(flowID int) (model.ScriptFlow, error) {
	var flow model.ScriptFlow
	err := postgresDB.Where("flow_id = ?", flowID).First(&flow).Error
	if err != nil {
		return model.ScriptFlow{}, uerr.NewError(err)
	}

	return flow, nil
}

func ListScriptFlows() ([]model.ScriptFlow, error) {
	var flows []model.ScriptFlow
	err := postgresDB.Order("flow_id ASC").Find(&flows).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return flows, nil
}

func SetScriptFlowEnabled(flowID int, enabled bool) error {
	result := postgresDB.Model(&model.ScriptFlow{}).Where("flow_id = ?", flowID).Update("is_enabled", enabled)
	if result.Error != nil {
		return uerr.NewError(result.Error)
	}

	return nil
}

func ReplaceFlowSteps(flowID int, steps []model.FlowStep) error {
	tx := postgresDB.Begin()
	if tx.Error != nil {
		return uerr.NewError(tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err := tx.Where("flow_id = ?", flowID).Delete(&model.FlowStep{}).Error
	if err != nil {
		tx.Rollback()
		return uerr.NewError(err)
	}

	for i := range steps {
		steps[i].FlowID = flowID
		if steps[i].StepOrder == 0 {
			steps[i].StepOrder = i + 1
		}
		if steps[i].TimeoutMs <= 0 {
			steps[i].TimeoutMs = 5000
		}
		if steps[i].FailureStrategy == "" {
			steps[i].FailureStrategy = "fail_close"
		}
		if !steps[i].IsEnabled {
			steps[i].IsEnabled = true
		}

		err = tx.Create(&steps[i]).Error
		if err != nil {
			tx.Rollback()
			return uerr.NewError(err)
		}
	}

	err = tx.Commit().Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func ListFlowSteps(flowID int) ([]model.FlowStep, error) {
	var steps []model.FlowStep
	err := postgresDB.Where("flow_id = ?", flowID).Order("step_order ASC").Find(&steps).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return steps, nil
}

func CreateFlowMount(mount *model.FlowMount) error {
	err := postgresDB.Create(mount).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func DeleteFlowMount(mountID int) error {
	err := postgresDB.Where("mount_id = ?", mountID).Delete(&model.FlowMount{}).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func ListFlowMountsByFlow(flowID int) ([]model.FlowMount, error) {
	var mounts []model.FlowMount
	err := postgresDB.Where("flow_id = ?", flowID).Order("mount_id ASC").Find(&mounts).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return mounts, nil
}

func ResolveFlowForExecution(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []ResolvedFlowStep, error) {
	mount, err := findMountedFlow(scope, eventKey, targetType, targetID)
	if err != nil {
		return model.ScriptFlow{}, nil, err
	}

	var flow model.ScriptFlow
	err = postgresDB.Where("flow_id = ? AND is_enabled = ?", mount.FlowID, true).First(&flow).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ScriptFlow{}, nil, ErrFlowNotMounted
		}
		return model.ScriptFlow{}, nil, uerr.NewError(err)
	}

	var steps []model.FlowStep
	err = postgresDB.Where("flow_id = ? AND is_enabled = ?", flow.FlowID, true).
		Order("step_order ASC").
		Find(&steps).Error
	if err != nil {
		return model.ScriptFlow{}, nil, uerr.NewError(err)
	}

	resolvedSteps := make([]ResolvedFlowStep, 0, len(steps))
	for _, step := range steps {
		scriptDef, getDefErr := GetScriptDefinitionByID(step.ScriptID)
		if getDefErr != nil {
			return model.ScriptFlow{}, nil, getDefErr
		}
		if !scriptDef.IsEnabled {
			return model.ScriptFlow{}, nil, uerr.NewError(errors.New("script is disabled: " + scriptDef.ScriptKey))
		}

		var version model.ScriptVersion
		if step.ScriptVersionID > 0 {
			version, err = GetScriptVersionByID(step.ScriptVersionID)
			if err != nil {
				return model.ScriptFlow{}, nil, err
			}
			if version.ScriptID != step.ScriptID {
				return model.ScriptFlow{}, nil, uerr.NewError(errors.New("script version does not match script"))
			}
		} else {
			version, err = GetActiveScriptVersion(step.ScriptID)
			if err != nil {
				return model.ScriptFlow{}, nil, err
			}
		}

		resolvedSteps = append(resolvedSteps, ResolvedFlowStep{
			Step:    step,
			Script:  scriptDef,
			Version: version,
		})
	}

	return flow, resolvedSteps, nil
}

func findMountedFlow(scope string, eventKey string, targetType string, targetID int) (model.FlowMount, error) {
	var mount model.FlowMount
	err := postgresDB.Where(
		"scope = ? AND event_key = ? AND target_type = ? AND target_id = ? AND is_enabled = ?",
		scope,
		eventKey,
		targetType,
		targetID,
		true,
	).First(&mount).Error
	if err == nil {
		return mount, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.FlowMount{}, uerr.NewError(err)
	}

	err = postgresDB.Where(
		"scope = ? AND event_key = ? AND target_type = ? AND target_id = ? AND is_enabled = ?",
		scope,
		eventKey,
		"global",
		0,
		true,
	).First(&mount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.FlowMount{}, ErrFlowNotMounted
		}
		return model.FlowMount{}, uerr.NewError(err)
	}

	return mount, nil
}
