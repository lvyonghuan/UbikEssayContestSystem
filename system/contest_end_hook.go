package system

import (
	"context"
	"errors"
	"fmt"
	"main/database/pgsql"
	"main/model"
	"main/util/log"
	"main/util/scriptflow"
	"path/filepath"
	"strings"
	"time"
)

var (
	getTracksByContestForEndFn = pgsql.GetTracksByContestID
	resolveFlowChainForEndFn   = pgsql.ResolveFlowChainForExecution

	executeScriptChainForEndFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		executor := scriptflow.NewExecutor(".", 5*time.Second, []string{"python3", "python", "bash", "sh", "node"})
		return executor.ExecuteChain(context.Background(), chain, input)
	}
)

func executeContestEndForContest(contestID int) error {
	tracks, err := getTracksByContestForEndFn(contestID)
	if err != nil {
		return err
	}

	var firstErr error
	for _, track := range tracks {
		if track.TrackID <= 0 {
			continue
		}
		if err := runContestEndHookForTrack(contestID, track.TrackID); err != nil {
			log.Logger.Warn("Run contest_end hook for track failed: " + err.Error())
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

func runContestEndHookForTrack(contestID int, trackID int) error {
	chains, err := resolveFlowChainForEndFn(scriptflow.ScopeSystem, scriptflow.EventContestEnd, contestID, trackID)
	if err != nil {
		if errors.Is(err, pgsql.ErrFlowNotMounted) {
			return nil
		}
		return err
	}

	payload := map[string]any{
		"phase":     "contest_end",
		"contestID": contestID,
		"trackID":   trackID,
	}

	for _, chain := range chains {
		result, err := executeResolvedContestEndFlow(contestID, trackID, chain.Flow, chain.Steps, payload)
		if err != nil {
			if errors.Is(err, scriptflow.ErrExecutionBlocked) {
				reason := strings.TrimSpace(result.Reason)
				if reason == "" {
					reason = "contest_end blocked by script flow"
				}
				return errors.New(reason)
			}
			return err
		}
		if !result.Allowed {
			reason := strings.TrimSpace(result.Reason)
			if reason == "" {
				reason = "contest_end blocked by script flow"
			}
			return errors.New(reason)
		}
	}

	return nil
}

func executeResolvedContestEndFlow(
	contestID int,
	trackID int,
	flow model.ScriptFlow,
	steps []pgsql.ResolvedFlowStep,
	payload map[string]any,
) (scriptflow.ChainResult, error) {
	chain := scriptflow.ChainConfig{
		Scope:    scriptflow.ScopeSystem,
		EventKey: scriptflow.EventContestEnd,
		FlowKey:  flow.FlowKey,
		Steps:    make([]scriptflow.StepConfig, 0, len(steps)),
	}

	for _, step := range steps {
		timeout := 5 * time.Second
		if step.Step.TimeoutMs > 0 {
			timeout = time.Duration(step.Step.TimeoutMs) * time.Millisecond
		}

		strategy := strings.TrimSpace(step.Step.FailureStrategy)
		if strategy == "" {
			strategy = "fail_close"
		}

		chain.Steps = append(chain.Steps, scriptflow.StepConfig{
			StepName:        step.Step.StepName,
			Interpreter:     step.Script.Interpreter,
			ScriptPath:      filepath.ToSlash(step.Version.RelativePath),
			Timeout:         timeout,
			FailureStrategy: strategy,
			InputTemplate:   step.Step.InputTemplate,
		})
	}

	result, err := executeScriptChainForEndFn(chain, scriptflow.ExecuteInput{
		Scope:    scriptflow.ScopeSystem,
		EventKey: scriptflow.EventContestEnd,
		FlowKey:  flow.FlowKey,
		TraceID:  fmt.Sprintf("contest_end_%d_%d_%d", contestID, trackID, time.Now().UnixNano()),
		NowUnix:  time.Now().Unix(),
		Context: map[string]any{
			"contestID": contestID,
			"trackID":   trackID,
		},
		Payload: payload,
	})
	if err != nil {
		return result, err
	}

	return result, nil
}
