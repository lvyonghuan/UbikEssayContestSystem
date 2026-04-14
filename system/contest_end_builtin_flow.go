package system

import (
	"context"
	"errors"
	"main/database/pgsql"
	"main/model"
	"main/util/log"
	"main/util/scriptflow"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

const (
	contestEndBuiltinRegenerateScriptKey = "contest_end_regenerate_review_results_builtin"
	contestEndBuiltinExportPDFScriptKey  = "contest_end_export_track_pdfs_builtin"

	contestEndBuiltinRegenerateStepKey = "builtin/contest_end/regenerate_review_results"
	contestEndBuiltinExportPDFStepKey  = "builtin/contest_end/export_track_pdfs"

	contestEndBuiltinFlowKey        = "contest_end_builtin_pipeline"
	contestEndBuiltinFlowName       = "Contest End Builtin Pipeline"
	contestEndBuiltinCreatorAdminID = 1
)

var (
	getScriptDefinitionByKeyForEndBuiltinFn   = pgsql.GetScriptDefinitionByKey
	createScriptDefinitionForEndBuiltinFn     = pgsql.CreateScriptDefinition
	updateScriptDefinitionForEndBuiltinFn     = pgsql.UpdateScriptDefinition
	listScriptVersionsForEndBuiltinFn         = pgsql.ListScriptVersions
	getNextScriptVersionNumberForEndBuiltinFn = pgsql.GetNextScriptVersionNumber
	createScriptVersionForEndBuiltinFn        = pgsql.CreateScriptVersion
	activateScriptVersionForEndBuiltinFn      = pgsql.ActivateScriptVersion
	listScriptFlowsForEndBuiltinFn            = pgsql.ListScriptFlows
	createScriptFlowForEndBuiltinFn           = pgsql.CreateScriptFlow
	setScriptFlowEnabledForEndBuiltinFn       = pgsql.SetScriptFlowEnabled
	getFlowMountByTargetForEndBuiltinFn       = pgsql.GetFlowMountByTarget
	createFlowMountForEndBuiltinFn            = pgsql.CreateFlowMount
	setFlowMountEnabledForEndBuiltinFn        = pgsql.SetFlowMountEnabled
	listFlowStepsForEndBuiltinFn              = pgsql.ListFlowSteps
	replaceFlowStepsForEndBuiltinFn           = pgsql.ReplaceFlowSteps
)

func newContestEndExecutor() *scriptflow.Executor {
	executor := scriptflow.NewExecutor(
		".",
		5*time.Second,
		[]string{"python3", "python", "bash", "sh", "node", scriptflow.InterpreterBuiltinGo},
	)
	executor.RegisterBuiltinStepHandlers(map[string]scriptflow.BuiltinStepHandler{
		contestEndBuiltinRegenerateStepKey: runContestEndRegenerateBuiltinStep,
		contestEndBuiltinExportPDFStepKey:  runContestEndExportPDFBuiltinStep,
	})

	return executor
}

func runContestEndRegenerateBuiltinStep(ctx context.Context, input scriptflow.ExecuteInput) (scriptflow.ExecuteOutput, error) {
	_, trackID, err := contestEndIDsFromExecuteInput(input)
	if err != nil {
		return scriptflow.ExecuteOutput{}, err
	}

	select {
	case <-ctx.Done():
		return scriptflow.ExecuteOutput{}, ctx.Err()
	default:
	}

	if err = regenerateTrackReviewResults(trackID); err != nil {
		return scriptflow.ExecuteOutput{}, err
	}

	return scriptflow.ExecuteOutput{Allow: true}, nil
}

func runContestEndExportPDFBuiltinStep(ctx context.Context, input scriptflow.ExecuteInput) (scriptflow.ExecuteOutput, error) {
	contestID, trackID, err := contestEndIDsFromExecuteInput(input)
	if err != nil {
		return scriptflow.ExecuteOutput{}, err
	}

	select {
	case <-ctx.Done():
		return scriptflow.ExecuteOutput{}, ctx.Err()
	default:
	}

	if err = exportTrackWorkPDFs(contestID, trackID); err != nil {
		return scriptflow.ExecuteOutput{}, err
	}

	return scriptflow.ExecuteOutput{Allow: true}, nil
}

func contestEndIDsFromExecuteInput(input scriptflow.ExecuteInput) (int, int, error) {
	contestID, ok := normalizeIntValue(input.Context["contestID"])
	if !ok || contestID <= 0 {
		return 0, 0, errors.New("contestID is required for contest_end builtin step")
	}

	trackID, ok := normalizeIntValue(input.Context["trackID"])
	if !ok || trackID <= 0 {
		return 0, 0, errors.New("trackID is required for contest_end builtin step")
	}

	return contestID, trackID, nil
}

func normalizeIntValue(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func initContestEndBuiltinFlow() error {
	regenerateScriptID, regenerateVersionID, err := ensureContestEndBuiltinScript(
		contestEndBuiltinRegenerateScriptKey,
		"Contest End Regenerate Review Results",
		"Built-in Go step: regenerate review_results for all works under a track at contest end.",
		contestEndBuiltinRegenerateStepKey,
	)
	if err != nil {
		return err
	}

	exportScriptID, exportVersionID, err := ensureContestEndBuiltinScript(
		contestEndBuiltinExportPDFScriptKey,
		"Contest End Export Track PDFs",
		"Built-in Go step: export each work file to PDF under files/pdfs/<contest>/<track>/<author>/.",
		contestEndBuiltinExportPDFStepKey,
	)
	if err != nil {
		return err
	}

	defaultFlowID, err := ensureContestEndBuiltinFlowDefinition()
	if err != nil {
		return err
	}

	mountedFlowID, err := ensureContestEndGlobalMount(defaultFlowID)
	if err != nil {
		return err
	}

	return ensureContestEndBuiltinFlowSteps(mountedFlowID, regenerateScriptID, regenerateVersionID, exportScriptID, exportVersionID)
}

func ensureContestEndBuiltinScript(scriptKey string, scriptName string, description string, stepKey string) (int, int, error) {
	scriptDef, err := getScriptDefinitionByKeyForEndBuiltinFn(scriptKey)
	if err != nil {
		if !isRecordNotFoundForBuiltin(err) {
			return 0, 0, uerr.ExtractError(err)
		}

		scriptDef = model.ScriptDefinition{
			ScriptKey:   scriptKey,
			ScriptName:  scriptName,
			Interpreter: scriptflow.InterpreterBuiltinGo,
			Description: description,
			IsEnabled:   true,
			Meta: map[string]any{
				"builtin": true,
				"engine":  "go",
			},
		}
		if err = createScriptDefinitionForEndBuiltinFn(&scriptDef); err != nil {
			return 0, 0, uerr.ExtractError(err)
		}
	} else {
		needsUpdate := false
		updated := model.ScriptDefinition{}
		if strings.TrimSpace(scriptDef.ScriptName) != scriptName {
			updated.ScriptName = scriptName
			needsUpdate = true
		}
		if strings.TrimSpace(scriptDef.Interpreter) != scriptflow.InterpreterBuiltinGo {
			updated.Interpreter = scriptflow.InterpreterBuiltinGo
			needsUpdate = true
		}
		if strings.TrimSpace(scriptDef.Description) != description {
			updated.Description = description
			needsUpdate = true
		}
		if !scriptDef.IsEnabled {
			updated.IsEnabled = true
			needsUpdate = true
		}
		if needsUpdate {
			if err = updateScriptDefinitionForEndBuiltinFn(scriptDef.ScriptID, &updated); err != nil {
				return 0, 0, uerr.ExtractError(err)
			}
		}
	}

	versions, err := listScriptVersionsForEndBuiltinFn(scriptDef.ScriptID)
	if err != nil {
		return 0, 0, uerr.ExtractError(err)
	}

	var targetVersion model.ScriptVersion
	for _, version := range versions {
		if filepath.ToSlash(strings.TrimSpace(version.RelativePath)) == stepKey {
			targetVersion = version
			break
		}
	}

	if targetVersion.VersionID == 0 {
		nextVersion, nextErr := getNextScriptVersionNumberForEndBuiltinFn(scriptDef.ScriptID)
		if nextErr != nil {
			return 0, 0, uerr.ExtractError(nextErr)
		}

		targetVersion = model.ScriptVersion{
			ScriptID:     scriptDef.ScriptID,
			VersionNum:   nextVersion,
			FileName:     filepath.Base(stepKey) + ".builtin",
			RelativePath: stepKey,
			Checksum:     "builtin:" + scriptDef.ScriptKey,
			IsActive:     false,
			CreatedBy:    contestEndBuiltinCreatorAdminID,
		}
		if err = createScriptVersionForEndBuiltinFn(&targetVersion); err != nil {
			return 0, 0, uerr.ExtractError(err)
		}
	}

	if !targetVersion.IsActive {
		if err = activateScriptVersionForEndBuiltinFn(scriptDef.ScriptID, targetVersion.VersionID); err != nil {
			return 0, 0, uerr.ExtractError(err)
		}
	}

	return scriptDef.ScriptID, targetVersion.VersionID, nil
}

func ensureContestEndBuiltinFlowDefinition() (int, error) {
	flows, err := listScriptFlowsForEndBuiltinFn()
	if err != nil {
		return 0, uerr.ExtractError(err)
	}

	for _, flow := range flows {
		if flow.FlowKey != contestEndBuiltinFlowKey {
			continue
		}
		if !flow.IsEnabled {
			if err = setScriptFlowEnabledForEndBuiltinFn(flow.FlowID, true); err != nil {
				return 0, uerr.ExtractError(err)
			}
		}
		return flow.FlowID, nil
	}

	flow := model.ScriptFlow{
		FlowKey:     contestEndBuiltinFlowKey,
		FlowName:    contestEndBuiltinFlowName,
		Description: "Built-in contest_end flow. Ensure track PDF export is executed in script flow pipeline.",
		IsEnabled:   true,
		Meta: map[string]any{
			"builtin": true,
			"event":   scriptflow.EventContestEnd,
			"scope":   scriptflow.ScopeSystem,
		},
	}

	if err = createScriptFlowForEndBuiltinFn(&flow); err != nil {
		return 0, uerr.ExtractError(err)
	}

	return flow.FlowID, nil
}

func ensureContestEndGlobalMount(defaultFlowID int) (int, error) {
	mount, err := getFlowMountByTargetForEndBuiltinFn(scriptflow.ScopeSystem, scriptflow.EventContestEnd, "global", 0)
	if err != nil {
		if !errors.Is(err, pgsql.ErrFlowNotMounted) {
			return 0, uerr.ExtractError(err)
		}

		mount = model.FlowMount{
			FlowID:     defaultFlowID,
			Scope:      scriptflow.ScopeSystem,
			EventKey:   scriptflow.EventContestEnd,
			TargetType: "global",
			TargetID:   0,
			IsEnabled:  true,
		}
		if err = createFlowMountForEndBuiltinFn(&mount); err != nil {
			return 0, uerr.ExtractError(err)
		}
		return defaultFlowID, nil
	}

	if !mount.IsEnabled {
		if err = setFlowMountEnabledForEndBuiltinFn(mount.MountID, true); err != nil {
			return 0, uerr.ExtractError(err)
		}
	}

	if mount.FlowID <= 0 {
		return 0, errors.New("contest_end global mount has invalid flow_id")
	}

	return mount.FlowID, nil
}

func ensureContestEndBuiltinFlowSteps(
	flowID int,
	regenerateScriptID int,
	regenerateVersionID int,
	exportScriptID int,
	exportVersionID int,
) error {
	steps, err := listFlowStepsForEndBuiltinFn(flowID)
	if err != nil {
		return uerr.ExtractError(err)
	}

	hasExport := false
	removedRegenerate := false
	merged := make([]model.FlowStep, 0, len(steps)+1)
	for _, step := range steps {
		if step.ScriptVersionID == regenerateVersionID || step.ScriptID == regenerateScriptID || strings.TrimSpace(step.StepName) == "builtin_regenerate_review_results" {
			removedRegenerate = true
			continue
		}
		if step.ScriptVersionID == exportVersionID || step.ScriptID == exportScriptID {
			hasExport = true
		}
		step.StepID = 0
		merged = append(merged, step)
	}

	insertedExport := false
	if !hasExport {
		merged = append([]model.FlowStep{{
			StepName:        "builtin_export_track_pdfs",
			ScriptID:        exportScriptID,
			ScriptVersionID: exportVersionID,
			TimeoutMs:       120000,
			FailureStrategy: "fail_close",
			InputTemplate: map[string]any{
				"handler": "export_track_pdfs",
			},
			IsEnabled: true,
		}}, merged...)
		insertedExport = true
	}

	if !insertedExport && !removedRegenerate {
		return nil
	}

	for i := range merged {
		merged[i].StepOrder = i + 1
	}

	if err = replaceFlowStepsForEndBuiltinFn(flowID, merged); err != nil {
		return uerr.ExtractError(err)
	}

	if log.Logger != nil {
		log.Logger.Warn(
			"contest_end builtin steps ensured in flow: flowID=" + strconv.Itoa(flowID) +
				" removedRegenerate=" + strconv.FormatBool(removedRegenerate) +
				" insertedExport=" + strconv.FormatBool(!hasExport),
		)
	}

	return nil
}

func isRecordNotFoundForBuiltin(err error) bool {
	if err == nil {
		return false
	}

	parsed := uerr.ExtractError(err)
	if errors.Is(parsed, gorm.ErrRecordNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}

	return strings.Contains(strings.ToLower(parsed.Error()), "record not found")
}
