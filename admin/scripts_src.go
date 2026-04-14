package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/scriptflow"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

var (
	createScriptDefinitionFn     = pgsql.CreateScriptDefinition
	updateScriptDefinitionFn     = pgsql.UpdateScriptDefinition
	getScriptDefinitionByIDFn    = pgsql.GetScriptDefinitionByID
	listScriptDefinitionsFn      = pgsql.ListScriptDefinitions
	setScriptDefinitionEnabledFn = pgsql.SetScriptDefinitionEnabled
	getNextScriptVersionNumberFn = pgsql.GetNextScriptVersionNumber
	createScriptVersionFn        = pgsql.CreateScriptVersion
	listScriptVersionsFn         = pgsql.ListScriptVersions
	activateScriptVersionFn      = pgsql.ActivateScriptVersion

	createScriptFlowFn     = pgsql.CreateScriptFlow
	updateScriptFlowFn     = pgsql.UpdateScriptFlow
	getScriptFlowByIDFn    = pgsql.GetScriptFlowByID
	listScriptFlowsFn      = pgsql.ListScriptFlows
	setScriptFlowEnabledFn = pgsql.SetScriptFlowEnabled
	replaceFlowStepsFn     = pgsql.ReplaceFlowSteps
	listFlowStepsFn        = pgsql.ListFlowSteps
	createFlowMountFn      = pgsql.CreateFlowMount
	deleteFlowMountFn      = pgsql.DeleteFlowMount
	listFlowMountsByFlowFn = pgsql.ListFlowMountsByFlow
	getScriptVersionByIDFn = pgsql.GetScriptVersionByID

	mkdirAllFn       = os.MkdirAll
	createFileFn     = os.Create
	openUploadFileFn = func(fileHeader *multipart.FileHeader) (multipart.File, error) {
		return fileHeader.Open()
	}
)

var (
	scriptKeyRegexp     = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	allowedInterpreters = map[string]struct{}{
		"python3":                       {},
		"python":                        {},
		"bash":                          {},
		"sh":                            {},
		"node":                          {},
		scriptflow.InterpreterBuiltinGo: {},
	}
)

func scriptSrcWarn(message string) {
	if log.Logger != nil {
		log.Logger.Warn(message)
	}
}

func newScriptSrcError(message string) error {
	err := errors.New(message)
	scriptSrcWarn("Script src error: " + err.Error())
	return err
}

func scriptSrcExtractError(err error) error {
	parsedErr := uerr.ExtractError(err)
	scriptSrcWarn("Script src error: " + parsedErr.Error())
	return parsedErr
}

func createScriptDefinitionSrc(adminID int, def *model.ScriptDefinition) error {
	def.ScriptKey = strings.TrimSpace(def.ScriptKey)
	def.ScriptName = strings.TrimSpace(def.ScriptName)
	def.Interpreter = strings.TrimSpace(def.Interpreter)
	if err := validateScriptDefinition(def); err != nil {
		return scriptSrcExtractError(err)
	}
	if def.Meta == nil {
		def.Meta = map[string]any{}
	}
	if !def.IsEnabled {
		def.IsEnabled = true
	}

	err := createScriptDefinitionFn(def)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.Scripts, _const.Create,
		genDetails([]string{"script_key", "script_id"}, []string{def.ScriptKey, strconv.Itoa(def.ScriptID)}))

	return nil
}

func updateScriptDefinitionSrc(adminID int, scriptID int, updated *model.ScriptDefinition) error {
	updated.ScriptName = strings.TrimSpace(updated.ScriptName)
	updated.Interpreter = strings.TrimSpace(updated.Interpreter)
	if updated.ScriptName == "" {
		return newScriptSrcError("script name is required")
	}
	if _, ok := allowedInterpreters[updated.Interpreter]; !ok {
		return newScriptSrcError("unsupported interpreter")
	}

	err := updateScriptDefinitionFn(scriptID, updated)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.Scripts, _const.Update,
		genDetails([]string{"script_id", "script_name"}, []string{strconv.Itoa(scriptID), updated.ScriptName}))

	return nil
}

func getScriptDefinitionByIDSrc(scriptID int) (model.ScriptDefinition, error) {
	def, err := getScriptDefinitionByIDFn(scriptID)
	if err != nil {
		return model.ScriptDefinition{}, scriptSrcExtractError(err)
	}
	return def, nil
}

func listScriptDefinitionsSrc() ([]model.ScriptDefinition, error) {
	defs, err := listScriptDefinitionsFn()
	if err != nil {
		return nil, scriptSrcExtractError(err)
	}
	return defs, nil
}

func setScriptDefinitionEnabledSrc(adminID int, scriptID int, enabled bool) error {
	err := setScriptDefinitionEnabledFn(scriptID, enabled)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	action := _const.Update
	createActionLogFn(adminID, _const.Scripts, action,
		genDetails([]string{"script_id", "is_enabled"}, []string{strconv.Itoa(scriptID), strconv.FormatBool(enabled)}))

	return nil
}

func uploadScriptVersionSrc(adminID int, scriptID int, fileHeader *multipart.FileHeader) (model.ScriptVersion, error) {
	def, err := getScriptDefinitionByIDFn(scriptID)
	if err != nil {
		return model.ScriptVersion{}, scriptSrcExtractError(err)
	}

	nextVersion, err := getNextScriptVersionNumberFn(scriptID)
	if err != nil {
		return model.ScriptVersion{}, scriptSrcExtractError(err)
	}

	safeName := filepath.Base(strings.TrimSpace(fileHeader.Filename))
	if safeName == "" || safeName == "." {
		return model.ScriptVersion{}, newScriptSrcError("invalid script file name")
	}

	versionDir := filepath.Join("scripts", def.ScriptKey, "v"+strconv.Itoa(nextVersion))
	if err = mkdirAllFn(versionDir, os.ModePerm); err != nil {
		wrappedErr := uerr.NewError(err)
		return model.ScriptVersion{}, scriptSrcExtractError(wrappedErr)
	}

	dstPath := filepath.Join(versionDir, safeName)
	srcFile, err := openUploadFileFn(fileHeader)
	if err != nil {
		wrappedErr := uerr.NewError(err)
		return model.ScriptVersion{}, scriptSrcExtractError(wrappedErr)
	}
	defer srcFile.Close()

	dstFile, err := createFileFn(dstPath)
	if err != nil {
		wrappedErr := uerr.NewError(err)
		return model.ScriptVersion{}, scriptSrcExtractError(wrappedErr)
	}

	hasher := sha256.New()
	_, err = io.Copy(io.MultiWriter(dstFile, hasher), srcFile)
	closeErr := dstFile.Close()
	if err != nil {
		_ = removeFn(dstPath)
		wrappedErr := uerr.NewError(err)
		return model.ScriptVersion{}, scriptSrcExtractError(wrappedErr)
	}
	if closeErr != nil {
		_ = removeFn(dstPath)
		wrappedErr := uerr.NewError(closeErr)
		return model.ScriptVersion{}, scriptSrcExtractError(wrappedErr)
	}

	version := model.ScriptVersion{
		ScriptID:     scriptID,
		VersionNum:   nextVersion,
		FileName:     safeName,
		RelativePath: filepath.ToSlash(dstPath),
		Checksum:     hex.EncodeToString(hasher.Sum(nil)),
		IsActive:     nextVersion == 1,
		CreatedBy:    adminID,
	}

	err = createScriptVersionFn(&version)
	if err != nil {
		_ = removeFn(dstPath)
		return model.ScriptVersion{}, scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.Scripts, _const.Update,
		genDetails([]string{"script_id", "version_id", "version_num"}, []string{strconv.Itoa(scriptID), strconv.Itoa(version.VersionID), strconv.Itoa(version.VersionNum)}))

	return version, nil
}

func listScriptVersionsSrc(scriptID int) ([]model.ScriptVersion, error) {
	versions, err := listScriptVersionsFn(scriptID)
	if err != nil {
		return nil, scriptSrcExtractError(err)
	}
	return versions, nil
}

func activateScriptVersionSrc(adminID int, scriptID int, versionID int) error {
	err := activateScriptVersionFn(scriptID, versionID)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.Scripts, _const.Update,
		genDetails([]string{"script_id", "version_id", "activate"}, []string{strconv.Itoa(scriptID), strconv.Itoa(versionID), "true"}))

	return nil
}

func createScriptFlowSrc(adminID int, flow *model.ScriptFlow) error {
	flow.FlowKey = strings.TrimSpace(flow.FlowKey)
	flow.FlowName = strings.TrimSpace(flow.FlowName)
	if flow.FlowKey == "" || flow.FlowName == "" {
		return newScriptSrcError("flow key and flow name are required")
	}
	if !scriptKeyRegexp.MatchString(flow.FlowKey) {
		return newScriptSrcError("flow key only supports letters, numbers, underscore and hyphen")
	}
	if flow.Meta == nil {
		flow.Meta = map[string]any{}
	}
	if !flow.IsEnabled {
		flow.IsEnabled = true
	}

	err := createScriptFlowFn(flow)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.ScriptFlows, _const.Create,
		genDetails([]string{"flow_key", "flow_id"}, []string{flow.FlowKey, strconv.Itoa(flow.FlowID)}))

	return nil
}

func updateScriptFlowSrc(adminID int, flowID int, updated *model.ScriptFlow) error {
	updated.FlowName = strings.TrimSpace(updated.FlowName)
	if updated.FlowName == "" {
		return newScriptSrcError("flow name is required")
	}

	err := updateScriptFlowFn(flowID, updated)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.ScriptFlows, _const.Update,
		genDetails([]string{"flow_id", "flow_name"}, []string{strconv.Itoa(flowID), updated.FlowName}))

	return nil
}

func getScriptFlowByIDSrc(flowID int) (model.ScriptFlow, error) {
	flow, err := getScriptFlowByIDFn(flowID)
	if err != nil {
		return model.ScriptFlow{}, scriptSrcExtractError(err)
	}
	return flow, nil
}

func listScriptFlowsSrc() ([]model.ScriptFlow, error) {
	flows, err := listScriptFlowsFn()
	if err != nil {
		return nil, scriptSrcExtractError(err)
	}
	return flows, nil
}

func setScriptFlowEnabledSrc(adminID int, flowID int, enabled bool) error {
	err := setScriptFlowEnabledFn(flowID, enabled)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.ScriptFlows, _const.Update,
		genDetails([]string{"flow_id", "is_enabled"}, []string{strconv.Itoa(flowID), strconv.FormatBool(enabled)}))

	return nil
}

func replaceFlowStepsSrc(adminID int, flowID int, steps []model.FlowStep) error {
	for i := range steps {
		if strings.TrimSpace(steps[i].StepName) == "" {
			return newScriptSrcError("step name is required")
		}
		if steps[i].ScriptID <= 0 {
			return newScriptSrcError("script id is required")
		}
		if steps[i].StepOrder == 0 {
			steps[i].StepOrder = i + 1
		}
		if steps[i].FailureStrategy == "" {
			steps[i].FailureStrategy = "fail_close"
		}
		if steps[i].TimeoutMs <= 0 {
			steps[i].TimeoutMs = 5000
		}
		if _, err := getScriptDefinitionByIDFn(steps[i].ScriptID); err != nil {
			return scriptSrcExtractError(err)
		}
		if steps[i].ScriptVersionID > 0 {
			version, err := getScriptVersionByIDFn(steps[i].ScriptVersionID)
			if err != nil {
				return scriptSrcExtractError(err)
			}
			if version.ScriptID != steps[i].ScriptID {
				return newScriptSrcError("script version does not belong to script")
			}
		}
	}

	err := replaceFlowStepsFn(flowID, steps)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.ScriptFlows, _const.Update,
		genDetails([]string{"flow_id", "steps_count"}, []string{strconv.Itoa(flowID), strconv.Itoa(len(steps))}))

	return nil
}

func listFlowStepsSrc(flowID int) ([]model.FlowStep, error) {
	steps, err := listFlowStepsFn(flowID)
	if err != nil {
		return nil, scriptSrcExtractError(err)
	}
	return steps, nil
}

func createFlowMountSrc(adminID int, mount *model.FlowMount) error {
	mount.Scope = strings.TrimSpace(mount.Scope)
	mount.EventKey = strings.TrimSpace(mount.EventKey)
	mount.TargetType = strings.TrimSpace(mount.TargetType)
	if mount.FlowID <= 0 || mount.Scope == "" || mount.EventKey == "" || mount.TargetType == "" {
		return newScriptSrcError("flow_id/scope/event_key/target_type are required")
	}
	if mount.TargetID < 0 {
		return newScriptSrcError("target_id must be >= 0")
	}
	if !mount.IsEnabled {
		mount.IsEnabled = true
	}

	if _, err := getScriptFlowByIDFn(mount.FlowID); err != nil {
		return scriptSrcExtractError(err)
	}

	err := createFlowMountFn(mount)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.ScriptFlows, _const.Create,
		genDetails([]string{"mount_id", "flow_id", "scope", "event_key"}, []string{strconv.Itoa(mount.MountID), strconv.Itoa(mount.FlowID), mount.Scope, mount.EventKey}))

	return nil
}

func deleteFlowMountSrc(adminID int, mountID int) error {
	err := deleteFlowMountFn(mountID)
	if err != nil {
		return scriptSrcExtractError(err)
	}

	createActionLogFn(adminID, _const.ScriptFlows, _const.Delete,
		genDetails([]string{"mount_id"}, []string{strconv.Itoa(mountID)}))

	return nil
}

func listFlowMountsByFlowSrc(flowID int) ([]model.FlowMount, error) {
	mounts, err := listFlowMountsByFlowFn(flowID)
	if err != nil {
		return nil, scriptSrcExtractError(err)
	}
	return mounts, nil
}

func validateScriptDefinition(def *model.ScriptDefinition) error {
	if def.ScriptKey == "" || def.ScriptName == "" {
		return newScriptSrcError("script key and script name are required")
	}
	if !scriptKeyRegexp.MatchString(def.ScriptKey) {
		return newScriptSrcError("script key only supports letters, numbers, underscore and hyphen")
	}
	if _, ok := allowedInterpreters[def.Interpreter]; !ok {
		return newScriptSrcError("unsupported interpreter")
	}
	return nil
}
