package admin

import (
	"main/model"
	"main/util/scriptflow"
	"strings"
	"testing"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

func TestReplaceFlowStepsSrcScriptNotFound(t *testing.T) {
	backupSrcHooks(t)

	createActionLogFn = func(adminID int, resource string, action string, details map[string]interface{}) {}
	getScriptDefinitionByIDFn = func(scriptID int) (model.ScriptDefinition, error) {
		return model.ScriptDefinition{}, gorm.ErrRecordNotFound
	}
	replaceFlowStepsFn = func(flowID int, steps []model.FlowStep) error {
		t.Fatal("replaceFlowStepsFn should not be called when script is missing")
		return nil
	}

	err := replaceFlowStepsSrc(1, 1, []model.FlowStep{{
		StepName:  "step-1",
		ScriptID:  1,
		IsEnabled: true,
	}})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != "script does not exist" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplaceFlowStepsSrcScriptVersionNotFound(t *testing.T) {
	backupSrcHooks(t)

	createActionLogFn = func(adminID int, resource string, action string, details map[string]interface{}) {}
	getScriptDefinitionByIDFn = func(scriptID int) (model.ScriptDefinition, error) {
		return model.ScriptDefinition{ScriptID: scriptID}, nil
	}
	getScriptVersionByIDFn = func(versionID int) (model.ScriptVersion, error) {
		return model.ScriptVersion{}, uerr.NewError(gorm.ErrRecordNotFound)
	}
	replaceFlowStepsFn = func(flowID int, steps []model.FlowStep) error {
		t.Fatal("replaceFlowStepsFn should not be called when script version is missing")
		return nil
	}

	err := replaceFlowStepsSrc(1, 1, []model.FlowStep{{
		StepName:        "step-1",
		ScriptID:        1,
		ScriptVersionID: 999,
		IsEnabled:       true,
	}})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != "script version does not exist" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateFlowMountSrcRejectsContestEndBuiltinOutsideContestEnd(t *testing.T) {
	backupSrcHooks(t)

	origGetScriptFlowByIDFn := getScriptFlowByIDFn
	origListFlowStepsFn := listFlowStepsFn
	origGetScriptDefinitionByIDFn := getScriptDefinitionByIDFn
	origGetScriptVersionByIDFn := getScriptVersionByIDFn
	origCreateFlowMountFn := createFlowMountFn
	t.Cleanup(func() {
		getScriptFlowByIDFn = origGetScriptFlowByIDFn
		listFlowStepsFn = origListFlowStepsFn
		getScriptDefinitionByIDFn = origGetScriptDefinitionByIDFn
		getScriptVersionByIDFn = origGetScriptVersionByIDFn
		createFlowMountFn = origCreateFlowMountFn
	})

	createActionLogFn = func(adminID int, resource string, action string, details map[string]interface{}) {}
	getScriptFlowByIDFn = func(flowID int) (model.ScriptFlow, error) {
		return model.ScriptFlow{FlowID: flowID, FlowKey: "flow"}, nil
	}
	listFlowStepsFn = func(flowID int) ([]model.FlowStep, error) {
		return []model.FlowStep{{StepID: 1, StepName: "builtin_export", ScriptID: 10, ScriptVersionID: 20, IsEnabled: true}}, nil
	}
	getScriptDefinitionByIDFn = func(scriptID int) (model.ScriptDefinition, error) {
		return model.ScriptDefinition{ScriptID: scriptID, ScriptKey: contestEndBuiltinExportPDFScriptKeyForMount, Interpreter: scriptflow.InterpreterBuiltinGo, IsEnabled: true}, nil
	}
	getScriptVersionByIDFn = func(versionID int) (model.ScriptVersion, error) {
		return model.ScriptVersion{VersionID: versionID, ScriptID: 10, RelativePath: "builtin/contest_end/export_track_pdfs"}, nil
	}
	createFlowMountFn = func(mount *model.FlowMount) error {
		t.Fatal("createFlowMountFn should not be called when mount is invalid")
		return nil
	}

	err := createFlowMountSrc(1, &model.FlowMount{
		FlowID:     1,
		Scope:      scriptflow.ScopeSubmission,
		EventKey:   scriptflow.EventFilePost,
		TargetType: "track",
		TargetID:   7,
		IsEnabled:  true,
	})
	if err == nil {
		t.Fatal("expected mount validation error")
	}
	if !strings.Contains(err.Error(), "contest_end builtin steps can only be mounted to system/contest_end") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateFlowMountSrcAllowsContestEndBuiltinOnContestEndEvent(t *testing.T) {
	backupSrcHooks(t)

	origGetScriptFlowByIDFn := getScriptFlowByIDFn
	origListFlowStepsFn := listFlowStepsFn
	origGetScriptDefinitionByIDFn := getScriptDefinitionByIDFn
	origGetScriptVersionByIDFn := getScriptVersionByIDFn
	origCreateFlowMountFn := createFlowMountFn
	t.Cleanup(func() {
		getScriptFlowByIDFn = origGetScriptFlowByIDFn
		listFlowStepsFn = origListFlowStepsFn
		getScriptDefinitionByIDFn = origGetScriptDefinitionByIDFn
		getScriptVersionByIDFn = origGetScriptVersionByIDFn
		createFlowMountFn = origCreateFlowMountFn
	})

	createActionLogFn = func(adminID int, resource string, action string, details map[string]interface{}) {}
	getScriptFlowByIDFn = func(flowID int) (model.ScriptFlow, error) {
		return model.ScriptFlow{FlowID: flowID, FlowKey: "flow"}, nil
	}
	listFlowStepsFn = func(flowID int) ([]model.FlowStep, error) {
		return []model.FlowStep{{StepID: 1, StepName: "builtin_export", ScriptID: 10, ScriptVersionID: 20, IsEnabled: true}}, nil
	}
	getScriptDefinitionByIDFn = func(scriptID int) (model.ScriptDefinition, error) {
		return model.ScriptDefinition{ScriptID: scriptID, ScriptKey: contestEndBuiltinExportPDFScriptKeyForMount, Interpreter: scriptflow.InterpreterBuiltinGo, IsEnabled: true}, nil
	}
	getScriptVersionByIDFn = func(versionID int) (model.ScriptVersion, error) {
		return model.ScriptVersion{VersionID: versionID, ScriptID: 10, RelativePath: "builtin/contest_end/export_track_pdfs"}, nil
	}
	createFlowMountFn = func(mount *model.FlowMount) error {
		mount.MountID = 99
		return nil
	}

	mount := &model.FlowMount{
		FlowID:     1,
		Scope:      scriptflow.ScopeSystem,
		EventKey:   scriptflow.EventContestEnd,
		TargetType: "global",
		TargetID:   0,
		IsEnabled:  true,
	}
	err := createFlowMountSrc(1, mount)
	if err != nil {
		t.Fatalf("expected mount creation success, got %v", err)
	}
	if mount.MountID != 99 {
		t.Fatalf("expected mount id to be assigned, got %d", mount.MountID)
	}
}
