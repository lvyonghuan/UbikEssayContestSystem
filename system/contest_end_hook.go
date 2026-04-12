package system

import (
	"context"
	"errors"
	"fmt"
	"io"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/scriptflow"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	getTracksByContestForEndFn    = pgsql.GetTracksByContestID
	resolveFlowChainForEndFn      = pgsql.ResolveFlowChainForExecution
	listReviewEventsForEndFn      = pgsql.ListReviewEventsByTrackID
	listReviewWorksForEndFn       = pgsql.GetReviewWorksByEvent
	listReviewsForEndFn           = pgsql.ListReviewsByWorkAndEvent
	listJudgeIDsForEndFn          = pgsql.ListJudgeIDsByReviewEvent
	deleteReviewResultsForEndFn   = pgsql.DeleteReviewResultsByEventID
	upsertReviewResultForEndFn    = pgsql.UpsertReviewResult
	listWorksByTrackForEndFn      = pgsql.GetWorksByTrackID
	resolveSubmissionFileForEndFn = resolveSubmissionFileForEnd
	convertDocxToPDFForEndFn      = convertDocxToPDF

	readDirForEndFn  = os.ReadDir
	mkdirAllForEndFn = os.MkdirAll

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
	if err := regenerateTrackReviewResults(trackID); err != nil {
		return err
	}

	if err := exportTrackWorkPDFs(contestID, trackID); err != nil {
		return err
	}

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

func regenerateTrackReviewResults(trackID int) error {
	events, err := listReviewEventsForEndFn(trackID)
	if err != nil {
		return err
	}

	for _, event := range events {
		if event.EventID <= 0 {
			continue
		}

		if err := deleteReviewResultsForEndFn(event.EventID); err != nil {
			return err
		}

		works, err := listReviewWorksForEndFn(event.EventID, 0, 1000000)
		if err != nil {
			return err
		}

		for _, work := range works {
			if work.WorkID <= 0 {
				continue
			}

			if err := regenerateReviewResultForWorkAndEvent(work.WorkID, event.EventID); err != nil {
				return err
			}
		}
	}

	return nil
}

func regenerateReviewResultForWorkAndEvent(workID int, eventID int) error {
	reviews, err := listReviewsForEndFn(workID, eventID)
	if err != nil {
		return err
	}

	judgeIDs, err := listJudgeIDsForEndFn(eventID)
	if err != nil {
		return err
	}

	totalScore := 0.0
	scoreCount := 0
	comments := make([]string, 0, len(reviews))
	judgeScores := map[string]float64{}

	for _, review := range reviews {
		score := toFloat64ForEnd(review.WorkReviews["judgeScore"])
		if score > 0 {
			totalScore += score
			scoreCount++
		}

		comment := strings.TrimSpace(toStringForEnd(review.WorkReviews["judgeComment"]))
		if comment != "" {
			comments = append(comments, comment)
		}

		judgeScores[strconv.Itoa(review.JudgeID)] = score
	}

	finalScore := 0.0
	if scoreCount > 0 {
		finalScore = totalScore / float64(scoreCount)
	}

	payload := map[string]any{
		"finalScore":         finalScore,
		"reviewCount":        len(reviews),
		"assignedJudgeCount": len(judgeIDs),
		"comments":           strings.Join(comments, "\n\n"),
		"judgeScores":        judgeScores,
		"generatedAt":        time.Now().UTC().Format(time.RFC3339),
	}

	_, err = upsertReviewResultForEndFn(workID, eventID, payload)
	return err
}

func exportTrackWorkPDFs(contestID int, trackID int) error {
	works, err := listWorksByTrackForEndFn(trackID)
	if err != nil {
		return err
	}

	for _, work := range works {
		if work.WorkID <= 0 || work.TrackID <= 0 || work.AuthorID <= 0 {
			continue
		}

		srcPath, err := resolveSubmissionFileForEndFn(work)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		if strings.ToLower(filepath.Ext(srcPath)) != ".docx" {
			continue
		}

		dstDir := filepath.Join(_const.FileRootPath, "pdfs", strconv.Itoa(contestID), strconv.Itoa(trackID), strconv.Itoa(work.AuthorID))
		if err := mkdirAllForEndFn(dstDir, os.ModePerm); err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, strconv.Itoa(work.WorkID)+".pdf")
		if err := convertDocxToPDFForEndFn(context.Background(), srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func resolveSubmissionFileForEnd(work model.Work) (string, error) {
	dstDir := filepath.Join(_const.SubmissionFileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	entries, err := readDirForEndFn(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", os.ErrNotExist
		}
		return "", err
	}

	prefix := strconv.Itoa(work.WorkID) + "."
	selectedName := ""
	selectedTime := time.Time{}
	hasDocx := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		isDocx := ext == ".docx"

		if isDocx {
			if !hasDocx || selectedName == "" || info.ModTime().After(selectedTime) {
				hasDocx = true
				selectedName = name
				selectedTime = info.ModTime()
			}
			continue
		}

		if hasDocx {
			continue
		}

		if selectedName == "" || info.ModTime().After(selectedTime) {
			selectedName = name
			selectedTime = info.ModTime()
		}
	}

	if selectedName == "" {
		return "", os.ErrNotExist
	}

	return filepath.Join(dstDir, selectedName), nil
}

func convertDocxToPDF(ctx context.Context, srcDocxPath string, dstPDFPath string) error {
	if strings.ToLower(filepath.Ext(srcDocxPath)) != ".docx" {
		return errors.New("source file must be .docx")
	}

	if strings.ToLower(filepath.Ext(dstPDFPath)) != ".pdf" {
		return errors.New("destination file must be .pdf")
	}

	if err := mkdirAllForEndFn(filepath.Dir(dstPDFPath), os.ModePerm); err != nil {
		return err
	}

	workDir, err := os.MkdirTemp(os.TempDir(), "docx-to-pdf-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	stepCtx := ctx
	if stepCtx == nil {
		stepCtx = context.Background()
	}
	stepCtx, cancel := context.WithTimeout(stepCtx, 60*time.Second)
	defer cancel()

	binaries := []string{"soffice", "libreoffice"}
	var convertErr error
	for _, binary := range binaries {
		cmd := exec.CommandContext(
			stepCtx,
			binary,
			"--headless",
			"--convert-to",
			"pdf",
			"--outdir",
			workDir,
			srcDocxPath,
		)

		output, err := cmd.CombinedOutput()
		if err == nil {
			convertErr = nil
			break
		}

		convertErr = fmt.Errorf("%s convert failed: %w, output: %s", binary, err, strings.TrimSpace(string(output)))
	}
	if convertErr != nil {
		return convertErr
	}

	generated := filepath.Join(workDir, strings.TrimSuffix(filepath.Base(srcDocxPath), filepath.Ext(srcDocxPath))+".pdf")
	if _, err := os.Stat(generated); err != nil {
		return fmt.Errorf("pdf not generated: %w", err)
	}

	return moveFileForEnd(generated, dstPDFPath)
}

func moveFileForEnd(srcPath string, dstPath string) error {
	err := os.Rename(srcPath, dstPath)
	if err == nil {
		return nil
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return os.Remove(srcPath)
}

func toFloat64ForEnd(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return parsed
		}
	}

	return 0
}

func toStringForEnd(value any) string {
	if value == nil {
		return ""
	}

	if text, ok := value.(string); ok {
		return text
	}

	return fmt.Sprintf("%v", value)
}
