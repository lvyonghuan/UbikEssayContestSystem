package submission

import (
    "path/filepath"
    "strings"
    "time"

    "main/database/pgsql"
    "main/util/scriptflow"
)

// makeStepConfigsFromResolved converts DB resolved steps into scriptflow.StepConfig
func makeStepConfigsFromResolved(steps []pgsql.ResolvedFlowStep) []scriptflow.StepConfig {
    if len(steps) == 0 {
        return nil
    }
    configs := make([]scriptflow.StepConfig, 0, len(steps))
    for _, step := range steps {
        timeout := 5 * time.Second
        if step.Step.TimeoutMs > 0 {
            timeout = time.Duration(step.Step.TimeoutMs) * time.Millisecond
        }
        strategy := strings.TrimSpace(step.Step.FailureStrategy)
        if strategy == "" {
            strategy = "fail_close"
        }

        configs = append(configs, scriptflow.StepConfig{
            StepName:        step.Step.StepName,
            Interpreter:     step.Script.Interpreter,
            ScriptPath:      filepath.ToSlash(step.Version.RelativePath),
            Timeout:         timeout,
            FailureStrategy: strategy,
            InputTemplate:   step.Step.InputTemplate,
        })
    }
    return configs
}
