package submission

import (
    "path/filepath"
    "testing"
    "time"

    "main/database/pgsql"
    "main/model"
)

func TestMakeStepConfigsFromResolved_Defaults(t *testing.T) {
    steps := []pgsql.ResolvedFlowStep{
        {
            Step: model.FlowStep{
                StepName:        "step1",
                TimeoutMs:       0,
                FailureStrategy: "",
                InputTemplate:   map[string]any{"a": "b"},
            },
            Script: model.ScriptDefinition{
                Interpreter: "python",
            },
            Version: model.ScriptVersion{
                RelativePath: "scripts\\my_script.py",
            },
        },
    }

    cfgs := makeStepConfigsFromResolved(steps)
    if len(cfgs) != 1 {
        t.Fatalf("expected 1 cfg, got %d", len(cfgs))
    }
    c := cfgs[0]
    if c.StepName != "step1" {
        t.Fatalf("unexpected StepName: %s", c.StepName)
    }
    if c.Interpreter != "python" {
        t.Fatalf("unexpected Interpreter: %s", c.Interpreter)
    }
    if c.ScriptPath != filepath.ToSlash("scripts\\my_script.py") {
        t.Fatalf("unexpected ScriptPath: %s", c.ScriptPath)
    }
    // check default timeout approximately 5s
    if c.Timeout < 4*time.Second || c.Timeout > 6*time.Second {
        t.Fatalf("unexpected Timeout: %v", c.Timeout)
    }
    if c.FailureStrategy != "fail_close" {
        t.Fatalf("unexpected FailureStrategy: %s", c.FailureStrategy)
    }
    if v, ok := c.InputTemplate["a"]; !ok || v != "b" {
        t.Fatalf("unexpected InputTemplate: %#v", c.InputTemplate)
    }
}

func TestMakeStepConfigsFromResolved_Customs(t *testing.T) {
    steps := []pgsql.ResolvedFlowStep{
        {
            Step: model.FlowStep{
                StepName:        "step2",
                TimeoutMs:       1234,
                FailureStrategy: "retry",
                InputTemplate:   map[string]any{"x": 1},
            },
            Script: model.ScriptDefinition{
                Interpreter: "bash",
            },
            Version: model.ScriptVersion{
                RelativePath: "scripts/run.sh",
            },
        },
    }

    cfgs := makeStepConfigsFromResolved(steps)
    if len(cfgs) != 1 {
        t.Fatalf("expected 1 cfg, got %d", len(cfgs))
    }
    c := cfgs[0]
    if c.Timeout != time.Duration(1234)*time.Millisecond {
        t.Fatalf("unexpected Timeout: %v", c.Timeout)
    }
    if c.FailureStrategy != "retry" {
        t.Fatalf("unexpected FailureStrategy: %s", c.FailureStrategy)
    }
    if c.ScriptPath != filepath.ToSlash("scripts/run.sh") {
        t.Fatalf("unexpected ScriptPath: %s", c.ScriptPath)
    }
}
