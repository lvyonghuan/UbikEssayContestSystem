package scriptflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func writeScriptFile(t *testing.T, dir string, allow bool) (string, string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		filePath := filepath.Join(dir, "hook.ps1")
		output := "{\"allow\":true,\"patch\":{\"score\":88}}"
		if !allow {
			output = "{\"allow\":false,\"message\":\"blocked\"}"
		}
		content := "Write-Output '" + output + "'"
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("write script failed: %v", err)
		}
		return "powershell", filepath.Base(filePath)
	}

	filePath := filepath.Join(dir, "hook.sh")
	output := "{\"allow\":true,\"patch\":{\"score\":88}}"
	if !allow {
		output = "{\"allow\":false,\"message\":\"blocked\"}"
	}
	content := "#!/bin/sh\necho '" + output + "'\n"
	if err := os.WriteFile(filePath, []byte(content), 0o755); err != nil {
		t.Fatalf("write script failed: %v", err)
	}
	return "sh", filepath.Base(filePath)
}

func TestExecuteChainSuccess(t *testing.T) {
	tmp := t.TempDir()
	interpreter, scriptPath := writeScriptFile(t, tmp, true)

	executor := NewExecutor(tmp, 3*time.Second, []string{interpreter})
	result, err := executor.ExecuteChain(context.Background(), ChainConfig{
		Scope:    ScopeSubmission,
		EventKey: EventSubmissionPre,
		FlowKey:  "f1",
		Steps: []StepConfig{{
			StepName:        "s1",
			Interpreter:     interpreter,
			ScriptPath:      scriptPath,
			FailureStrategy: "fail_close",
			Timeout:         2 * time.Second,
		}},
	}, ExecuteInput{Scope: ScopeSubmission, EventKey: EventSubmissionPre})
	if err != nil {
		t.Fatalf("ExecuteChain should succeed: %v", err)
	}
	if !result.Allowed {
		t.Fatal("result should be allowed")
	}
	if result.Patch["score"] != float64(88) {
		t.Fatalf("unexpected patch: %+v", result.Patch)
	}
}

func TestExecuteChainBlocked(t *testing.T) {
	tmp := t.TempDir()
	interpreter, scriptPath := writeScriptFile(t, tmp, false)

	executor := NewExecutor(tmp, 3*time.Second, []string{interpreter})
	result, err := executor.ExecuteChain(context.Background(), ChainConfig{
		Scope:    ScopeSubmission,
		EventKey: EventSubmissionPre,
		FlowKey:  "f1",
		Steps: []StepConfig{{
			StepName:        "s1",
			Interpreter:     interpreter,
			ScriptPath:      scriptPath,
			FailureStrategy: "fail_close",
			Timeout:         2 * time.Second,
		}},
	}, ExecuteInput{Scope: ScopeSubmission, EventKey: EventSubmissionPre})
	if !errors.Is(err, ErrExecutionBlocked) {
		t.Fatalf("expected ErrExecutionBlocked, got %v", err)
	}
	if result.Allowed {
		t.Fatal("result should be blocked")
	}
}

func TestExecuteChainFailOpen(t *testing.T) {
	tmp := t.TempDir()
	executor := NewExecutor(tmp, 1*time.Second, []string{"powershell", "sh"})
	result, err := executor.ExecuteChain(context.Background(), ChainConfig{
		Scope:    ScopeSubmission,
		EventKey: EventSubmissionPre,
		FlowKey:  "f1",
		Steps: []StepConfig{{
			StepName:        "s1",
			Interpreter:     "sh",
			ScriptPath:      "not_exists.sh",
			FailureStrategy: "fail_open",
			Timeout:         time.Second,
		}},
	}, ExecuteInput{Scope: ScopeSubmission, EventKey: EventSubmissionPre})
	if err != nil {
		t.Fatalf("fail_open should continue without returning error: %v", err)
	}
	if !result.Allowed {
		t.Fatal("result should remain allowed for fail_open")
	}
}

func TestExecuteChainBuiltinStepSuccess(t *testing.T) {
	executor := NewExecutor(".", 2*time.Second, []string{InterpreterBuiltinGo})
	executor.RegisterBuiltinStepHandler("builtin/test/ok", func(ctx context.Context, input ExecuteInput) (ExecuteOutput, error) {
		return ExecuteOutput{
			Allow: true,
			Patch: map[string]any{"ok": true},
		}, nil
	})

	result, err := executor.ExecuteChain(context.Background(), ChainConfig{
		Scope:    ScopeSystem,
		EventKey: EventContestEnd,
		FlowKey:  "flow_builtin",
		Steps: []StepConfig{{
			StepName:        "builtin_step",
			Interpreter:     InterpreterBuiltinGo,
			ScriptPath:      "builtin/test/ok",
			FailureStrategy: "fail_close",
			Timeout:         2 * time.Second,
		}},
	}, ExecuteInput{Scope: ScopeSystem, EventKey: EventContestEnd})
	if err != nil {
		t.Fatalf("builtin step should succeed: %v", err)
	}
	if !result.Allowed {
		t.Fatal("builtin step result should be allowed")
	}
	if got, ok := result.Patch["ok"].(bool); !ok || !got {
		t.Fatalf("unexpected builtin patch: %+v", result.Patch)
	}
}

func TestExecuteChainBuiltinStepMissingHandler(t *testing.T) {
	executor := NewExecutor(".", 2*time.Second, []string{InterpreterBuiltinGo})

	result, err := executor.ExecuteChain(context.Background(), ChainConfig{
		Scope:    ScopeSystem,
		EventKey: EventContestEnd,
		FlowKey:  "flow_builtin",
		Steps: []StepConfig{{
			StepName:        "builtin_step",
			Interpreter:     InterpreterBuiltinGo,
			ScriptPath:      "builtin/test/missing",
			FailureStrategy: "fail_close",
			Timeout:         2 * time.Second,
		}},
	}, ExecuteInput{Scope: ScopeSystem, EventKey: EventContestEnd})
	if err == nil {
		t.Fatal("expected missing builtin handler to return error")
	}
	if result.Allowed {
		t.Fatal("result should be blocked when builtin handler is missing")
	}
}
