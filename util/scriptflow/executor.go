package scriptflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

var (
	ErrExecutionBlocked = errors.New("script flow blocked the request")
)

type BuiltinStepHandler func(ctx context.Context, input ExecuteInput) (ExecuteOutput, error)

type Executor struct {
	baseDir             string
	defaultTimeout      time.Duration
	allowedInterpreters map[string]struct{}
	builtinStepHandlers map[string]BuiltinStepHandler
}

func NewExecutor(baseDir string, defaultTimeout time.Duration, allowedInterpreters []string) *Executor {
	allowed := map[string]struct{}{}
	for _, interpreter := range allowedInterpreters {
		allowed[strings.TrimSpace(interpreter)] = struct{}{}
	}

	if defaultTimeout <= 0 {
		defaultTimeout = 5 * time.Second
	}

	return &Executor{
		baseDir:             baseDir,
		defaultTimeout:      defaultTimeout,
		allowedInterpreters: allowed,
		builtinStepHandlers: map[string]BuiltinStepHandler{},
	}
}

func (e *Executor) RegisterBuiltinStepHandler(stepKey string, handler BuiltinStepHandler) {
	if e == nil || handler == nil {
		return
	}

	key := filepath.ToSlash(strings.TrimSpace(stepKey))
	if key == "" {
		return
	}

	e.builtinStepHandlers[key] = handler
}

func (e *Executor) RegisterBuiltinStepHandlers(handlers map[string]BuiltinStepHandler) {
	for stepKey, handler := range handlers {
		e.RegisterBuiltinStepHandler(stepKey, handler)
	}
}

func (e *Executor) ExecuteChain(ctx context.Context, chain ChainConfig, input ExecuteInput) (ChainResult, error) {
	result := ChainResult{
		Allowed: true,
		Patch:   map[string]any{},
		Steps:   make([]StepResult, 0, len(chain.Steps)),
	}

	if len(chain.Steps) == 0 {
		return result, nil
	}

	for _, step := range chain.Steps {
		attempts := 1
		if step.FailureStrategy == "retry" {
			if step.RetryCount > 0 {
				attempts = step.RetryCount + 1
			} else {
				attempts = 2
			}
		}

		var (
			stepResult StepResult
			output     ExecuteOutput
			err        error
		)

		for attempt := 1; attempt <= attempts; attempt++ {
			stepResult, output, err = e.executeStep(ctx, step, input)
			if err == nil {
				break
			}
			if attempt == attempts {
				break
			}
		}

		if err != nil {
			result.Steps = append(result.Steps, stepResult)
			if step.FailureStrategy == "fail_open" {
				continue
			}
			result.Allowed = false
			result.Reason = err.Error()
			return result, err
		}

		result.Steps = append(result.Steps, stepResult)
		mergePatch(result.Patch, output.Patch)

		if !output.Allow {
			result.Allowed = false
			result.Reason = output.Message
			if result.Reason == "" {
				result.Reason = "blocked by script step"
			}
			return result, ErrExecutionBlocked
		}
	}

	if len(result.Patch) == 0 {
		result.Patch = nil
	}

	return result, nil
}

func (e *Executor) executeStep(ctx context.Context, step StepConfig, input ExecuteInput) (StepResult, ExecuteOutput, error) {
	started := time.Now()
	stepResult := StepResult{
		StepName: step.StepName,
		Success:  false,
	}

	interpreter := strings.TrimSpace(step.Interpreter)
	if interpreter == "" {
		stepResult.DurationMs = time.Since(started).Milliseconds()
		stepResult.Message = "interpreter is required"
		return stepResult, ExecuteOutput{}, uerr.NewError(errors.New(stepResult.Message))
	}

	if len(e.allowedInterpreters) > 0 {
		if _, ok := e.allowedInterpreters[interpreter]; !ok {
			stepResult.DurationMs = time.Since(started).Milliseconds()
			stepResult.Message = "interpreter is not allowed: " + interpreter
			return stepResult, ExecuteOutput{}, uerr.NewError(errors.New(stepResult.Message))
		}
	}

	scriptRef := strings.TrimSpace(step.ScriptPath)
	if scriptRef == "" {
		stepResult.DurationMs = time.Since(started).Milliseconds()
		stepResult.Message = "script path is required"
		return stepResult, ExecuteOutput{}, uerr.NewError(errors.New(stepResult.Message))
	}

	timeout := step.Timeout
	if timeout <= 0 {
		timeout = e.defaultTimeout
	}
	stepCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request := ExecuteInput{
		Scope:    input.Scope,
		EventKey: input.EventKey,
		FlowKey:  input.FlowKey,
		TraceID:  input.TraceID,
		NowUnix:  input.NowUnix,
		Context:  copyMap(input.Context),
		Payload:  copyMap(input.Payload),
	}
	if request.Context == nil {
		request.Context = map[string]any{}
	}
	request.Context["stepInput"] = step.InputTemplate

	if interpreter == InterpreterBuiltinGo {
		output, err := e.executeBuiltinStep(stepCtx, scriptRef, request)
		stepResult.DurationMs = time.Since(started).Milliseconds()
		if err != nil {
			stepResult.Message = err.Error()
			return stepResult, ExecuteOutput{}, err
		}

		stepResult.Success = true
		stepResult.Message = output.Message
		stepResult.Patch = output.Patch
		return stepResult, output, nil
	}

	scriptPath, err := e.resolveScriptPath(scriptRef)
	if err != nil {
		stepResult.DurationMs = time.Since(started).Milliseconds()
		stepResult.Message = err.Error()
		return stepResult, ExecuteOutput{}, err
	}

	stdinData, err := json.Marshal(request)
	if err != nil {
		stepResult.DurationMs = time.Since(started).Milliseconds()
		stepResult.Message = "marshal step input failed"
		return stepResult, ExecuteOutput{}, uerr.NewError(err)
	}

	cmd := exec.CommandContext(stepCtx, interpreter, scriptPath)
	cmd.Stdin = bytes.NewReader(stdinData)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	stepResult.DurationMs = time.Since(started).Milliseconds()
	if runErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = runErr.Error()
		}
		stepResult.Message = msg
		return stepResult, ExecuteOutput{}, uerr.NewError(errors.New(msg))
	}

	output, err := parseOutput(stdout.String())
	if err != nil {
		stepResult.Message = err.Error()
		return stepResult, ExecuteOutput{}, err
	}

	stepResult.Success = true
	stepResult.Message = output.Message
	stepResult.Patch = output.Patch
	return stepResult, output, nil
}

func (e *Executor) executeBuiltinStep(ctx context.Context, stepKey string, input ExecuteInput) (ExecuteOutput, error) {
	key := filepath.ToSlash(strings.TrimSpace(stepKey))
	if key == "" {
		return ExecuteOutput{}, uerr.NewError(errors.New("builtin step key is required"))
	}

	handler, ok := e.builtinStepHandlers[key]
	if !ok {
		return ExecuteOutput{}, uerr.NewError(errors.New("builtin step handler not found: " + key))
	}

	output, err := handler(ctx, input)
	if err != nil {
		return ExecuteOutput{}, uerr.NewError(err)
	}

	if !output.Allow && strings.TrimSpace(output.Message) == "" {
		output.Allow = true
	}

	return output, nil
}

func (e *Executor) resolveScriptPath(relativePath string) (string, error) {
	cleanPath := filepath.Clean(relativePath)
	if cleanPath == "." || cleanPath == string(filepath.Separator) {
		return "", uerr.NewError(errors.New("invalid script path"))
	}

	baseAbs, err := filepath.Abs(e.baseDir)
	if err != nil {
		return "", uerr.NewError(err)
	}

	scriptAbs := filepath.Clean(filepath.Join(baseAbs, filepath.FromSlash(cleanPath)))
	prefix := baseAbs + string(filepath.Separator)
	if scriptAbs != baseAbs && !strings.HasPrefix(scriptAbs, prefix) {
		return "", uerr.NewError(errors.New("script path escapes base directory"))
	}

	info, statErr := os.Stat(scriptAbs)
	if statErr != nil {
		return "", uerr.NewError(statErr)
	}
	if info.IsDir() {
		return "", uerr.NewError(errors.New("script path points to a directory"))
	}

	return scriptAbs, nil
}

func parseOutput(raw string) (ExecuteOutput, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ExecuteOutput{Allow: true}, nil
	}

	type rawOutput struct {
		Allow   *bool          `json:"allow"`
		Message string         `json:"message,omitempty"`
		Patch   map[string]any `json:"patch,omitempty"`
		Metrics map[string]any `json:"metrics,omitempty"`
	}

	var out rawOutput
	err := json.Unmarshal([]byte(trimmed), &out)
	if err != nil {
		preview := trimmed
		if len(preview) > 256 {
			preview = preview[:256]
		}
		return ExecuteOutput{}, uerr.NewError(fmt.Errorf("invalid script output json: %w; stdout=%s", err, preview))
	}

	allow := true
	if out.Allow != nil {
		allow = *out.Allow
	}

	return ExecuteOutput{
		Allow:   allow,
		Message: out.Message,
		Patch:   out.Patch,
		Metrics: out.Metrics,
	}, nil
}

func copyMap(origin map[string]any) map[string]any {
	if origin == nil {
		return nil
	}
	cloned := make(map[string]any, len(origin))
	for k, v := range origin {
		cloned[k] = v
	}
	return cloned
}

func mergePatch(dst map[string]any, src map[string]any) {
	if dst == nil || len(src) == 0 {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}
