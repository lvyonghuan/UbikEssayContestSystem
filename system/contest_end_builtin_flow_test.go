package system

import (
	"main/model"
	"testing"
)

func backupContestEndBuiltinFlowDeps(t *testing.T) {
	t.Helper()

	origListFlowStepsForEndBuiltinFn := listFlowStepsForEndBuiltinFn
	origReplaceFlowStepsForEndBuiltinFn := replaceFlowStepsForEndBuiltinFn

	t.Cleanup(func() {
		listFlowStepsForEndBuiltinFn = origListFlowStepsForEndBuiltinFn
		replaceFlowStepsForEndBuiltinFn = origReplaceFlowStepsForEndBuiltinFn
	})
}

func TestEnsureContestEndBuiltinFlowStepsRemovesRegenerateStep(t *testing.T) {
	backupContestEndBuiltinFlowDeps(t)

	const (
		flowID             = 100
		regenerateScriptID = 1
		regenerateVersion  = 11
		exportScriptID     = 2
		exportVersion      = 22
	)

	listFlowStepsForEndBuiltinFn = func(inputFlowID int) ([]model.FlowStep, error) {
		if inputFlowID != flowID {
			t.Fatalf("unexpected flowID: %d", inputFlowID)
		}
		return []model.FlowStep{
			{StepID: 1, StepOrder: 1, StepName: "builtin_regenerate_review_results", ScriptID: regenerateScriptID, ScriptVersionID: regenerateVersion, TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
			{StepID: 2, StepOrder: 2, StepName: "custom_step", ScriptID: 9, ScriptVersionID: 99, TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
			{StepID: 3, StepOrder: 3, StepName: "builtin_export_track_pdfs", ScriptID: exportScriptID, ScriptVersionID: exportVersion, TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
		}, nil
	}

	replaceCalls := 0
	replaced := make([]model.FlowStep, 0)
	replaceFlowStepsForEndBuiltinFn = func(inputFlowID int, steps []model.FlowStep) error {
		replaceCalls++
		if inputFlowID != flowID {
			t.Fatalf("unexpected flowID for replace: %d", inputFlowID)
		}
		replaced = append(replaced, steps...)
		return nil
	}

	err := ensureContestEndBuiltinFlowSteps(flowID, regenerateScriptID, regenerateVersion, exportScriptID, exportVersion)
	if err != nil {
		t.Fatalf("ensureContestEndBuiltinFlowSteps failed: %v", err)
	}
	if replaceCalls != 1 {
		t.Fatalf("expected one replace call, got %d", replaceCalls)
	}
	if len(replaced) != 2 {
		t.Fatalf("expected 2 steps after removing regenerate, got %d", len(replaced))
	}
	if replaced[0].StepName != "custom_step" || replaced[1].StepName != "builtin_export_track_pdfs" {
		t.Fatalf("unexpected step order after cleanup: %+v", replaced)
	}
	if replaced[0].StepOrder != 1 || replaced[1].StepOrder != 2 {
		t.Fatalf("expected reordered step numbers, got %+v", replaced)
	}
}

func TestEnsureContestEndBuiltinFlowStepsInsertsExportWhenMissing(t *testing.T) {
	backupContestEndBuiltinFlowDeps(t)

	const (
		flowID             = 200
		regenerateScriptID = 1
		regenerateVersion  = 11
		exportScriptID     = 2
		exportVersion      = 22
	)

	listFlowStepsForEndBuiltinFn = func(inputFlowID int) ([]model.FlowStep, error) {
		return []model.FlowStep{{
			StepID:          1,
			StepOrder:       1,
			StepName:        "custom_step",
			ScriptID:        9,
			ScriptVersionID: 99,
			TimeoutMs:       5000,
			FailureStrategy: "fail_close",
			IsEnabled:       true,
		}}, nil
	}

	replaced := make([]model.FlowStep, 0)
	replaceFlowStepsForEndBuiltinFn = func(inputFlowID int, steps []model.FlowStep) error {
		replaced = append(replaced, steps...)
		return nil
	}

	err := ensureContestEndBuiltinFlowSteps(flowID, regenerateScriptID, regenerateVersion, exportScriptID, exportVersion)
	if err != nil {
		t.Fatalf("ensureContestEndBuiltinFlowSteps failed: %v", err)
	}
	if len(replaced) != 2 {
		t.Fatalf("expected 2 steps after inserting export, got %d", len(replaced))
	}
	if replaced[0].StepName != "builtin_export_track_pdfs" {
		t.Fatalf("export step should be prepended, got %+v", replaced)
	}
	if replaced[0].ScriptID != exportScriptID || replaced[0].ScriptVersionID != exportVersion {
		t.Fatalf("unexpected export binding: %+v", replaced[0])
	}
}

func TestEnsureContestEndBuiltinFlowStepsNoopWhenAlreadyValid(t *testing.T) {
	backupContestEndBuiltinFlowDeps(t)

	const (
		flowID             = 300
		regenerateScriptID = 1
		regenerateVersion  = 11
		exportScriptID     = 2
		exportVersion      = 22
	)

	listFlowStepsForEndBuiltinFn = func(inputFlowID int) ([]model.FlowStep, error) {
		return []model.FlowStep{
			{StepID: 1, StepOrder: 1, StepName: "custom_step", ScriptID: 9, ScriptVersionID: 99, TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
			{StepID: 2, StepOrder: 2, StepName: "builtin_export_track_pdfs", ScriptID: exportScriptID, ScriptVersionID: exportVersion, TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
		}, nil
	}

	replaceCalled := false
	replaceFlowStepsForEndBuiltinFn = func(inputFlowID int, steps []model.FlowStep) error {
		replaceCalled = true
		return nil
	}

	err := ensureContestEndBuiltinFlowSteps(flowID, regenerateScriptID, regenerateVersion, exportScriptID, exportVersion)
	if err != nil {
		t.Fatalf("ensureContestEndBuiltinFlowSteps failed: %v", err)
	}
	if replaceCalled {
		t.Fatal("replace should not be called when flow already has export and no regenerate")
	}
}
