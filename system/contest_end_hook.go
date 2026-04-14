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
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

var (
	getTracksByContestForEndFn     = pgsql.GetTracksByContestID
	getTrackByIDForEndFn           = pgsql.GetTrackByID
	getContestEndExecutionForEndFn = func(contestID int, trackID int) (contestEndExecutionState, error) {
		track, err := getTrackByIDForEndFn(trackID)
		if err != nil {
			return contestEndExecutionState{}, err
		}
		if contestID > 0 && track.ContestID > 0 && track.ContestID != contestID {
			return contestEndExecutionState{}, errors.New("track " + strconv.Itoa(trackID) + " is not under contest " + strconv.Itoa(contestID))
		}
		return contestEndStateFromTrack(track), nil
	}
	markTrackContestEndRunningForEndFn = pgsql.MarkTrackContestEndRunning
	markContestEndRunningForEndFn      = func(contestID int, trackID int, triggerSource string) error {
		return markTrackContestEndRunningForEndFn(trackID, triggerSource)
	}
	markTrackContestEndSuccessForEndFn = pgsql.MarkTrackContestEndSuccess
	markContestEndSuccessForEndFn      = func(contestID int, trackID int, triggerSource string) error {
		return markTrackContestEndSuccessForEndFn(trackID, triggerSource)
	}
	markTrackContestEndFailedForEndFn = pgsql.MarkTrackContestEndFailed
	markContestEndFailedForEndFn      = func(contestID int, trackID int, triggerSource string, lastError string) error {
		return markTrackContestEndFailedForEndFn(trackID, triggerSource, lastError)
	}
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

	readDirForEndFn    = os.ReadDir
	mkdirAllForEndFn   = os.MkdirAll
	renameFileForEndFn = os.Rename
	openFileForEndFn   = os.Open
	createFileForEndFn = os.Create
	copyFileForEndFn   = io.Copy

	executeScriptChainForEndFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		executor := newContestEndExecutor()
		return executor.ExecuteChain(context.Background(), chain, input)
	}
)

func executeContestEndForContest(contestID int) error {
	return executeContestEndForContestWithSource(contestID, contestEndTriggerSourceSystem)
}

func executeContestEndForContestWithSource(contestID int, triggerSource string) error {
	tracks, err := getTracksByContestForEndFn(contestID)
	if err != nil {
		return err
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_contest_start: contestID=%d triggerSource=%s tracks=%d",
			contestID,
			strings.TrimSpace(triggerSource),
			len(tracks),
		))
	}

	var firstErr error
	runCount := 0
	failedCount := 0
	skippedInvalidTrackCount := 0
	for _, track := range tracks {
		if track.TrackID <= 0 {
			skippedInvalidTrackCount++
			if log.Logger != nil {
				log.Logger.Debug(fmt.Sprintf(
					"contest_end_track_skip_invalid: contestID=%d trackID=%d",
					contestID,
					track.TrackID,
				))
			}
			continue
		}
		runCount++
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_track_dispatch: contestID=%d trackID=%d triggerSource=%s",
				contestID,
				track.TrackID,
				strings.TrimSpace(triggerSource),
			))
		}
		if err := runContestEndHookForTrackWithSource(contestID, track.TrackID, triggerSource); err != nil {
			failedCount++
			log.Logger.Warn("Run contest_end hook for track failed: " + err.Error())
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_contest_finish: contestID=%d triggerSource=%s dispatched=%d failed=%d skippedInvalid=%d",
			contestID,
			strings.TrimSpace(triggerSource),
			runCount,
			failedCount,
			skippedInvalidTrackCount,
		))
	}

	return firstErr
}

func runContestEndHookForTrack(contestID int, trackID int) error {
	return runContestEndHookForTrackWithSource(contestID, trackID, contestEndTriggerSourceSystem)
}

func runContestEndHookForTrackWithSource(contestID int, trackID int, triggerSource string) error {
	shouldExecute, state, hasState, err := shouldRunContestEndForTrackForEnd(contestID, trackID)
	if err != nil {
		return err
	}
	if log.Logger != nil {
		if hasState {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_track_gate: contestID=%d trackID=%d triggerSource=%s shouldExecute=%t %s",
				contestID,
				trackID,
				strings.TrimSpace(triggerSource),
				shouldExecute,
				formatContestEndStateForLog(state),
			))
		} else {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_track_gate: contestID=%d trackID=%d triggerSource=%s shouldExecute=%t state=not-found",
				contestID,
				trackID,
				strings.TrimSpace(triggerSource),
				shouldExecute,
			))
		}
	}
	if !shouldExecute {
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_track_skip_by_state: contestID=%d trackID=%d triggerSource=%s",
				contestID,
				trackID,
				strings.TrimSpace(triggerSource),
			))
		}
		return nil
	}

	err = markContestEndRunningForEndFn(contestID, trackID, triggerSource)
	if err != nil {
		return err
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_track_mark_running: contestID=%d trackID=%d triggerSource=%s",
			contestID,
			trackID,
			strings.TrimSpace(triggerSource),
		))
	}

	chains, err := resolveFlowChainForEndFn(scriptflow.ScopeSystem, scriptflow.EventContestEnd, contestID, trackID)
	if err != nil {
		if errors.Is(err, pgsql.ErrFlowNotMounted) {
			if log.Logger != nil {
				log.Logger.Warn(
					"contest_end_flow_not_mounted: skip flow execution and mark success; contestID=" + strconv.Itoa(contestID) +
						" trackID=" + strconv.Itoa(trackID) +
						" triggerSource=" + strings.TrimSpace(triggerSource),
				)
			}
			return markContestEndSuccessForEndFn(contestID, trackID, triggerSource)
		}
		markContestEndFailedIfNeeded(contestID, trackID, triggerSource, err)
		return err
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_flow_chain_resolved: contestID=%d trackID=%d chains=%d flows=%s",
			contestID,
			trackID,
			len(chains),
			summarizeFlowChainsForEnd(chains),
		))
	}

	payload := map[string]any{
		"phase":     "contest_end",
		"contestID": contestID,
		"trackID":   trackID,
	}

	for _, chain := range chains {
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_flow_chain_start: contestID=%d trackID=%d flowKey=%s targetType=%s targetID=%d steps=%d",
				contestID,
				trackID,
				chain.Flow.FlowKey,
				chain.TargetType,
				chain.TargetID,
				len(chain.Steps),
			))
		}
		result, err := executeResolvedContestEndFlow(contestID, trackID, chain.Flow, chain.Steps, payload)
		if err != nil {
			if errors.Is(err, scriptflow.ErrExecutionBlocked) {
				reason := strings.TrimSpace(result.Reason)
				if reason == "" {
					reason = "contest_end blocked by script flow"
				}
				blockedErr := uerr.NewError(errors.New(reason))
				markContestEndFailedIfNeeded(contestID, trackID, triggerSource, blockedErr)
				return blockedErr
			}
			markContestEndFailedIfNeeded(contestID, trackID, triggerSource, err)
			return err
		}
		if !result.Allowed {
			reason := strings.TrimSpace(result.Reason)
			if reason == "" {
				reason = "contest_end blocked by script flow"
			}
			blockedErr := uerr.NewError(errors.New(reason))
			markContestEndFailedIfNeeded(contestID, trackID, triggerSource, blockedErr)
			return blockedErr
		}
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_flow_chain_finish: contestID=%d trackID=%d flowKey=%s allowed=%t reason=%q",
				contestID,
				trackID,
				chain.Flow.FlowKey,
				result.Allowed,
				strings.TrimSpace(result.Reason),
			))
		}
	}

	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_track_mark_success: contestID=%d trackID=%d triggerSource=%s",
			contestID,
			trackID,
			strings.TrimSpace(triggerSource),
		))
	}
	return markContestEndSuccessForEndFn(contestID, trackID, triggerSource)
}

func shouldRunContestEndForTrackForEnd(contestID int, trackID int) (bool, contestEndExecutionState, bool, error) {
	state, err := getContestEndExecutionForEndFn(contestID, trackID)
	if err != nil {
		if isContestEndExecutionNotFound(err) {
			return true, contestEndExecutionState{}, false, nil
		}
		return false, contestEndExecutionState{}, false, err
	}

	return shouldExecuteContestEndByState(state, time.Now().UTC()), state, true, nil
}

func markContestEndFailedIfNeeded(contestID int, trackID int, triggerSource string, runErr error) {
	if runErr == nil {
		return
	}
	if log.Logger != nil {
		log.Logger.Warn(fmt.Sprintf(
			"contest_end_track_mark_failed: contestID=%d trackID=%d triggerSource=%s err=%s",
			contestID,
			trackID,
			strings.TrimSpace(triggerSource),
			runErr.Error(),
		))
	}

	err := markContestEndFailedForEndFn(contestID, trackID, triggerSource, runErr.Error())
	if err != nil && log.Logger != nil {
		log.Logger.Warn("Mark contest_end failed status error: " + err.Error())
	}
}

func executeResolvedContestEndFlow(
	contestID int,
	trackID int,
	flow model.ScriptFlow,
	steps []pgsql.ResolvedFlowStep,
	payload map[string]any,
) (scriptflow.ChainResult, error) {
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_flow_execute_start: contestID=%d trackID=%d flowKey=%s stepCount=%d stepNames=%s",
			contestID,
			trackID,
			flow.FlowKey,
			len(steps),
			summarizeFlowStepNamesForEnd(steps),
		))
	}
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
		if log.Logger != nil {
			log.Logger.Warn(fmt.Sprintf(
				"contest_end_flow_execute_fail: contestID=%d trackID=%d flowKey=%s err=%s allowed=%t reason=%q",
				contestID,
				trackID,
				flow.FlowKey,
				err.Error(),
				result.Allowed,
				strings.TrimSpace(result.Reason),
			))
		}
		return result, err
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_flow_execute_finish: contestID=%d trackID=%d flowKey=%s allowed=%t reason=%q patchKeys=%s",
			contestID,
			trackID,
			flow.FlowKey,
			result.Allowed,
			strings.TrimSpace(result.Reason),
			summarizePatchKeysForEnd(result.Patch),
		))
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
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_pdf_export_start: contestID=%d trackID=%d works=%d",
			contestID,
			trackID,
			len(works),
		))
	}

	exportedCount := 0
	missingSourceCount := 0
	skippedNonDocxCount := 0

	for _, work := range works {
		if work.WorkID <= 0 || work.TrackID <= 0 || work.AuthorID <= 0 {
			if log.Logger != nil {
				log.Logger.Debug(fmt.Sprintf(
					"contest_end_pdf_export_skip_invalid_work: contestID=%d trackID=%d workID=%d authorID=%d",
					contestID,
					trackID,
					work.WorkID,
					work.AuthorID,
				))
			}
			continue
		}

		srcPath, err := resolveSubmissionFileForEndFn(work)
		if err != nil {
			if os.IsNotExist(err) {
				missingSourceCount++
				if log.Logger != nil {
					log.Logger.Debug(fmt.Sprintf(
						"contest_end_pdf_export_missing_source: contestID=%d trackID=%d workID=%d authorID=%d",
						contestID,
						trackID,
						work.WorkID,
						work.AuthorID,
					))
				}
				continue
			}
			return err
		}
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_pdf_export_source_resolved: contestID=%d trackID=%d workID=%d authorID=%d src=%s",
				contestID,
				trackID,
				work.WorkID,
				work.AuthorID,
				srcPath,
			))
		}

		if strings.ToLower(filepath.Ext(srcPath)) != ".docx" {
			skippedNonDocxCount++
			if log.Logger != nil {
				log.Logger.Debug(fmt.Sprintf(
					"contest_end_pdf_export_skip_non_docx: contestID=%d trackID=%d workID=%d src=%s",
					contestID,
					trackID,
					work.WorkID,
					srcPath,
				))
			}
			continue
		}

		dstDir := filepath.Join(_const.FileRootPath, "pdfs", strconv.Itoa(contestID), strconv.Itoa(trackID), strconv.Itoa(work.AuthorID))
		if err := mkdirAllForEndFn(dstDir, os.ModePerm); err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, strconv.Itoa(work.WorkID)+".pdf")
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_pdf_convert_start: contestID=%d trackID=%d workID=%d src=%s dst=%s",
				contestID,
				trackID,
				work.WorkID,
				srcPath,
				dstPath,
			))
		}
		if err := convertDocxToPDFForEndFn(context.Background(), srcPath, dstPath); err != nil {
			return err
		}
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_pdf_convert_success: contestID=%d trackID=%d workID=%d dst=%s",
				contestID,
				trackID,
				work.WorkID,
				dstPath,
			))
		}

		exportedCount++
	}

	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_pdf_export_summary: contestID=%d trackID=%d works=%d exported=%d missing_source=%d skipped_non_docx=%d",
			contestID,
			trackID,
			len(works),
			exportedCount,
			missingSourceCount,
			skippedNonDocxCount,
		))

		if len(works) > 0 && exportedCount == 0 && (missingSourceCount > 0 || skippedNonDocxCount > 0) {
			log.Logger.Warn(fmt.Sprintf(
				"contest_end_pdf_export_zero_output: contestID=%d trackID=%d works=%d missing_source=%d skipped_non_docx=%d",
				contestID,
				trackID,
				len(works),
				missingSourceCount,
				skippedNonDocxCount,
			))
		}
	}

	return nil
}

func resolveSubmissionFileForEnd(work model.Work) (string, error) {
	dstDir := filepath.Join(_const.SubmissionFileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	entries, err := readDirForEndFn(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			if log.Logger != nil {
				log.Logger.Warn(fmt.Sprintf(
					"contest_end_submission_dir_not_found: trackID=%d authorID=%d workID=%d dir=%s",
					work.TrackID,
					work.AuthorID,
					work.WorkID,
					dstDir,
				))
			}
			return "", os.ErrNotExist
		}
		return "", uerr.NewError(err)
	}

	prefix := strconv.Itoa(work.WorkID) + "."
	selectedName := ""
	selectedTime := time.Time{}
	hasDocx := false
	prefixMatches := 0
	scannedFileNames := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		scannedFileNames = append(scannedFileNames, name)
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		prefixMatches++

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
		if log.Logger != nil {
			log.Logger.Warn(fmt.Sprintf(
				"contest_end_submission_file_not_found: trackID=%d authorID=%d workID=%d dir=%s prefix=%s scanned=%d prefix_matches=%d entries=%s",
				work.TrackID,
				work.AuthorID,
				work.WorkID,
				dstDir,
				prefix,
				len(scannedFileNames),
				prefixMatches,
				summarizeEntryNamesForEnd(scannedFileNames, 20),
			))
		}
		return "", os.ErrNotExist
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_submission_file_selected: trackID=%d authorID=%d workID=%d selected=%s dir=%s scanned=%d prefix_matches=%d prefer_docx=%t",
			work.TrackID,
			work.AuthorID,
			work.WorkID,
			selectedName,
			dstDir,
			len(scannedFileNames),
			prefixMatches,
			hasDocx,
		))
	}

	return filepath.Join(dstDir, selectedName), nil
}

func summarizeEntryNamesForEnd(names []string, limit int) string {
	if len(names) == 0 {
		return "[]"
	}
	if limit <= 0 || len(names) <= limit {
		return strings.Join(names, ",")
	}
	return strings.Join(names[:limit], ",") + ",..."
}

func buildOfficeConverterCandidates() []string {
	seen := make(map[string]struct{})
	binaries := make([]string, 0, 5)

	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if _, exists := seen[candidate]; exists {
			return
		}
		seen[candidate] = struct{}{}
		binaries = append(binaries, candidate)
	}

	add(os.Getenv("UBIK_LIBREOFFICE_BIN"))
	add("soffice")
	add("libreoffice")

	if runtime.GOOS == "windows" {
		if programFiles := strings.TrimSpace(os.Getenv("ProgramFiles")); programFiles != "" {
			add(filepath.Join(programFiles, "LibreOffice", "program", "soffice.exe"))
		}
		if programFilesX86 := strings.TrimSpace(os.Getenv("ProgramFiles(x86)")); programFilesX86 != "" {
			add(filepath.Join(programFilesX86, "LibreOffice", "program", "soffice.exe"))
		}
	}

	return binaries
}

func trimCommandOutput(output []byte) string {
	text := strings.TrimSpace(string(output))
	if len(text) <= 500 {
		return text
	}
	return text[:500] + "..."
}

func convertDocxToPDF(ctx context.Context, srcDocxPath string, dstPDFPath string) error {
	if strings.ToLower(filepath.Ext(srcDocxPath)) != ".docx" {
		return uerr.NewError(errors.New("source file must be .docx"))
	}

	if strings.ToLower(filepath.Ext(dstPDFPath)) != ".pdf" {
		return uerr.NewError(errors.New("destination file must be .pdf"))
	}

	if err := mkdirAllForEndFn(filepath.Dir(dstPDFPath), os.ModePerm); err != nil {
		return uerr.NewError(err)
	}

	workDir, err := os.MkdirTemp(os.TempDir(), "docx-to-pdf-*")
	if err != nil {
		return uerr.NewError(err)
	}
	defer os.RemoveAll(workDir)

	stepCtx := ctx
	if stepCtx == nil {
		stepCtx = context.Background()
	}
	stepCtx, cancel := context.WithTimeout(stepCtx, 60*time.Second)
	defer cancel()

	binaries := buildOfficeConverterCandidates()
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_pdf_converter_candidates: src=%s dst=%s candidates=%s",
			srcDocxPath,
			dstPDFPath,
			strings.Join(binaries, ","),
		))
	}
	var convertErr error
	attemptErrors := make([]string, 0, len(binaries))
	for idx, binary := range binaries {
		resolvedBinary := binary
		if !strings.ContainsAny(binary, `\\/`) {
			if lookedUpBinary, lookErr := exec.LookPath(binary); lookErr == nil {
				resolvedBinary = lookedUpBinary
			}
		}
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_pdf_convert_attempt: attempt=%d/%d binary=%s resolved=%s src=%s dst=%s",
				idx+1,
				len(binaries),
				binary,
				resolvedBinary,
				srcDocxPath,
				dstPDFPath,
			))
		}

		cmd := exec.CommandContext(
			stepCtx,
			resolvedBinary,
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
			if log.Logger != nil {
				log.Logger.Debug(fmt.Sprintf(
					"contest_end_pdf_convert_attempt_success: attempt=%d/%d binary=%s",
					idx+1,
					len(binaries),
					resolvedBinary,
				))
			}
			break
		}

		trimmedOutput := trimCommandOutput(output)
		convertErr = fmt.Errorf("%s convert failed: %w, output: %s", resolvedBinary, err, trimmedOutput)
		attemptErrors = append(attemptErrors, convertErr.Error())
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_pdf_convert_attempt_fail: attempt=%d/%d binary=%s err=%s",
				idx+1,
				len(binaries),
				resolvedBinary,
				convertErr.Error(),
			))
		}
	}
	if convertErr != nil {
		pathHasLibreOffice := strings.Contains(strings.ToLower(os.Getenv("PATH")), "libreoffice")
		if len(attemptErrors) == 0 {
			attemptErrors = append(attemptErrors, "no converter candidate available")
		}
		return uerr.NewError(fmt.Errorf("libreoffice convert failed after %d attempts (PATH has libreoffice=%t): %s", len(attemptErrors), pathHasLibreOffice, strings.Join(attemptErrors, " | ")))
	}

	generated := filepath.Join(workDir, strings.TrimSuffix(filepath.Base(srcDocxPath), filepath.Ext(srcDocxPath))+".pdf")
	if _, err := os.Stat(generated); err != nil {
		return uerr.NewError(fmt.Errorf("pdf not generated: %w", err))
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_pdf_generated_tmp: src=%s generated=%s",
			srcDocxPath,
			generated,
		))
	}

	return moveFileForEnd(generated, dstPDFPath)
}

func summarizeFlowChainsForEnd(chains []pgsql.ResolvedFlowChain) string {
	if len(chains) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(chains))
	for _, chain := range chains {
		parts = append(parts, fmt.Sprintf("%s(%s:%d,steps=%d)", chain.Flow.FlowKey, chain.TargetType, chain.TargetID, len(chain.Steps)))
	}
	return strings.Join(parts, ",")
}

func summarizeFlowStepNamesForEnd(steps []pgsql.ResolvedFlowStep) string {
	if len(steps) == 0 {
		return "[]"
	}
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		name := strings.TrimSpace(step.Step.StepName)
		if name == "" {
			name = "step_" + strconv.Itoa(step.Step.StepOrder)
		}
		names = append(names, name)
	}
	return strings.Join(names, ",")
}

func summarizePatchKeysForEnd(patch map[string]any) string {
	if len(patch) == 0 {
		return "[]"
	}
	keys := make([]string, 0, len(patch))
	for key := range patch {
		keys = append(keys, key)
	}
	return strings.Join(keys, ",")
}

func moveFileForEnd(srcPath string, dstPath string) error {
	err := renameFileForEndFn(srcPath, dstPath)
	if err == nil {
		return nil
	}

	srcFile, err := openFileForEndFn(srcPath)
	if err != nil {
		return uerr.NewError(err)
	}
	defer srcFile.Close()

	dstFile, err := createFileForEndFn(dstPath)
	if err != nil {
		return uerr.NewError(err)
	}
	defer dstFile.Close()

	if _, err = copyFileForEndFn(dstFile, srcFile); err != nil {
		return uerr.NewError(err)
	}

	return nil
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
