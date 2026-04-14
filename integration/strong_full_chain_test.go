package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"main/admin"
	"main/conf"
	"main/database/pgsql"
	"main/database/redis"
	"main/model"
	"main/submission"
	"main/util/log"
	"main/util/password"
	"main/util/token"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type strongEnv struct {
	repoRoot string
	config   *conf.Config
}

type testInputFile struct {
	AbsPath        string `json:"absPath"`
	RelPath        string `json:"relPath"`
	Category       string `json:"category"`
	Ext            string `json:"ext"`
	AuthorRaw      string `json:"authorRaw"`
	TitleRaw       string `json:"titleRaw"`
	ParseFallback  bool   `json:"parseFallback"`
	NameNoExt      string `json:"nameNoExt"`
	OriginalBase   string `json:"originalBase"`
	CategoryTrack  int    `json:"categoryTrack"`
	AuthorIdentity string `json:"authorIdentity"`
}

type fileRunResult struct {
	Scenario       string `json:"scenario"`
	FilePath       string `json:"filePath"`
	Category       string `json:"category"`
	Extension      string `json:"extension"`
	AuthorRaw      string `json:"authorRaw"`
	TitleRaw       string `json:"titleRaw"`
	ParseFallback  bool   `json:"parseFallback"`
	TrackID        int    `json:"trackID"`
	AuthorID       int    `json:"authorID"`
	SubmissionCode int    `json:"submissionCode"`
	SubmissionMsg  string `json:"submissionMsg,omitempty"`
	WorkID         int    `json:"workID,omitempty"`
	UploadCode     int    `json:"uploadCode,omitempty"`
	UploadMsg      string `json:"uploadMsg,omitempty"`
	SavedDocx      string `json:"savedDocx,omitempty"`
	WordCount      int    `json:"wordCount,omitempty"`
}

type scenarioSummary struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type fullChainReport struct {
	GeneratedAt      string            `json:"generatedAt"`
	RunID            string            `json:"runID"`
	InputFileCount   int               `json:"inputFileCount"`
	ContestIDs       map[string]int    `json:"contestIDs"`
	TrackIDs         map[string]int    `json:"trackIDs"`
	ScenarioSummary  []scenarioSummary `json:"scenarioSummary"`
	FileRunResults   []fileRunResult   `json:"fileRunResults"`
	CategoryCounters map[string]int    `json:"categoryCounters"`
	ExtCounters      map[string]int    `json:"extCounters"`
}

type authorSession struct {
	AuthorName string
	AuthorID   int
	Token      string
}

type apiCallResult struct {
	Code int
	Msg  any
	Body string
}

var (
	strongSetupOnce sync.Once
	strongSetupEnv  strongEnv
	strongSetupErr  error
)

const (
	categoryMidShort = "\u4e2d\u77ed\u7bc7\u5f81\u6587\u6587\u4ef6"
	categoryScript   = "\u5267\u672c\u5f81\u6587\u6587\u4ef6"
	categoryUltra    = "\u8d85\u77ed\u7bc7\u5f81\u6587\u6587\u4ef6"
)

func TestStrongDepsFullChainAllFiles(t *testing.T) {
	env := mustSetupStrongEnv(t)

	adminServer := httptest.NewServer(admin.BuildAdminRouter())
	t.Cleanup(adminServer.Close)

	submissionServer := httptest.NewServer(submission.BuildSubmissionRouter())
	t.Cleanup(submissionServer.Close)

	adminBaseURL := adminServer.URL
	submissionBaseURL := submissionServer.URL

	client := &http.Client{Timeout: 120 * time.Second}

	runID := time.Now().Format("20060102_150405")
	adminPassword := "StrongE2E!2026"
	adminToken := mustAdminLogin(t, client, adminBaseURL, adminPassword)

	files := mustScanAllTestFiles(t, env.repoRoot)
	if len(files) == 0 {
		t.Fatal("tests_files has no files")
	}

	report := fullChainReport{
		GeneratedAt:      time.Now().Format(time.RFC3339),
		RunID:            runID,
		InputFileCount:   len(files),
		ContestIDs:       map[string]int{},
		TrackIDs:         map[string]int{},
		FileRunResults:   make([]fileRunResult, 0, len(files)+32),
		CategoryCounters: map[string]int{},
		ExtCounters:      map[string]int{},
		ScenarioSummary:  make([]scenarioSummary, 0, 8),
	}

	for _, f := range files {
		report.CategoryCounters[f.Category]++
		report.ExtCounters[f.Ext]++
	}
	// S1: ongoing contest with all files
	contestID, trackByCategory := mustCreateContestAndTracks(
		t,
		client,
		adminBaseURL,
		adminToken,
		"e2e_full_all_files_"+runID,
		time.Now().Add(-24*time.Hour),
		time.Now().Add(7*24*time.Hour),
	)
	report.ContestIDs["S1_ongoing_full"] = contestID
	for category, trackID := range trackByCategory {
		report.TrackIDs["S1_"+category] = trackID
	}

	mustInstallSubmissionHooks(
		t,
		client,
		adminBaseURL,
		adminToken,
		env.repoRoot,
		runID,
		[]int{trackByCategory[categoryMidShort], trackByCategory[categoryScript], trackByCategory[categoryUltra]},
	)

	authorsS1 := map[string]*authorSession{}
	for i := range files {
		file := files[i]
		trackID := trackByCategory[file.Category]
		if trackID == 0 {
			trackID = trackByCategory[categoryMidShort]
		}

		session := mustEnsureAuthorSession(t, client, submissionBaseURL, authorsS1, file.AuthorIdentity, runID+"_S1")
		res := runOneFileThroughSubmission(
			t,
			client,
			submissionBaseURL,
			env.repoRoot,
			"S1_ongoing_full",
			file,
			session,
			trackID,
		)
		report.FileRunResults = append(report.FileRunResults, res)
	}
	report.ScenarioSummary = append(report.ScenarioSummary, scenarioSummary{
		Name:   "S1_ongoing_full",
		Passed: len(report.FileRunResults) >= len(files),
		Detail: "all files under tests_files were executed through submission and upload pipeline",
	})
	// S2: contest not started
	futureContestID, futureTracks := mustCreateContestAndTracks(
		t,
		client,
		adminBaseURL,
		adminToken,
		"e2e_not_started_"+runID,
		time.Now().Add(24*time.Hour),
		time.Now().Add(10*24*time.Hour),
	)
	report.ContestIDs["S2_not_started"] = futureContestID
	futureSession := mustEnsureAuthorSession(t, client, submissionBaseURL, map[string]*authorSession{}, "not_started_author", runID+"_S2")
	futureCode, futureMsg, _ := mustSubmitWork(
		t,
		client,
		submissionBaseURL,
		futureSession,
		futureTracks[categoryMidShort],
		"not started check",
		map[string]any{"scenario": "S2_not_started"},
	)
	notStartedPass := futureCode == 500 && strings.Contains(strings.ToLower(futureMsg), "not started")
	report.ScenarioSummary = append(report.ScenarioSummary, scenarioSummary{
		Name:   "S2_not_started",
		Passed: notStartedPass,
		Detail: fmt.Sprintf("submission code=%d msg=%s", futureCode, futureMsg),
	})
	// S3: contest already ended
	pastContestID, pastTracks := mustCreateContestAndTracks(
		t,
		client,
		adminBaseURL,
		adminToken,
		"e2e_already_ended_"+runID,
		time.Now().Add(-10*24*time.Hour),
		time.Now().Add(-24*time.Hour),
	)
	report.ContestIDs["S3_ended"] = pastContestID
	pastSession := mustEnsureAuthorSession(t, client, submissionBaseURL, map[string]*authorSession{}, "ended_author", runID+"_S3")
	pastCode, pastMsg, _ := mustSubmitWork(
		t,
		client,
		submissionBaseURL,
		pastSession,
		pastTracks[categoryMidShort],
		"ended check",
		map[string]any{"scenario": "S3_ended"},
	)
	endedPass := pastCode == 500 && strings.Contains(strings.ToLower(pastMsg), "already ended")
	report.ScenarioSummary = append(report.ScenarioSummary, scenarioSummary{
		Name:   "S3_ended",
		Passed: endedPass,
		Detail: fmt.Sprintf("submission code=%d msg=%s", pastCode, pastMsg),
	})

	// S4/S5: 鏉堝彞绗傞梽鎰┾偓浣稿灩闂勩倕鎮楃紒褏鐢婚幎鏇狀焾
	limitContestID, limitTracks := mustCreateContestAndTracks(
		t,
		client,
		adminBaseURL,
		adminToken,
		"e2e_limit_delete_retry_"+runID,
		time.Now().Add(-24*time.Hour),
		time.Now().Add(7*24*time.Hour),
	)
	report.ContestIDs["S4_S5_limit_delete_retry"] = limitContestID
	limitTrackID := limitTracks[categoryMidShort]
	mustInstallSubmissionHooks(t, client, adminBaseURL, adminToken, env.repoRoot, runID+"_S4", []int{limitTrackID})

	supported := pickSupportedDocx(files, 5)
	if len(supported) < 5 {
		t.Fatalf("need at least 5 .docx files for S4/S5, got %d", len(supported))
	}

	limitSession := mustEnsureAuthorSession(t, client, submissionBaseURL, map[string]*authorSession{}, "limit_author", runID+"_S4")
	createdWorks := make([]struct {
		WorkID int
		Title  string
		File   testInputFile
	}, 0, 4)

	limitBlocked := false
	for i := 0; i < 4; i++ {
		title := fmt.Sprintf("limit case %d", i+1)
		code, msg, workID := mustSubmitWork(
			t,
			client,
			submissionBaseURL,
			limitSession,
			limitTrackID,
			title,
			map[string]any{"scenario": "S4_limit"},
		)
		if i < 3 {
			if code != 200 {
				t.Fatalf("S4 first 3 submissions must pass, idx=%d code=%d msg=%s", i+1, code, msg)
			}
			mustUploadAndVerify(t, client, submissionBaseURL, env.repoRoot, limitSession, workID, limitTrackID, supported[i])
			createdWorks = append(createdWorks, struct {
				WorkID int
				Title  string
				File   testInputFile
			}{WorkID: workID, Title: title, File: supported[i]})
		} else {
			if code == 500 && strings.Contains(strings.ToLower(msg), "at most 3") {
				limitBlocked = true
			}
		}
	}

	if !limitBlocked {
		t.Fatalf("S4 expected 4th submission to be blocked by limit script")
	}

	if len(createdWorks) == 0 {
		t.Fatalf("S5 requires at least one created work")
	}
	mustDeleteWork(t, client, submissionBaseURL, limitSession, limitTrackID, createdWorks[0].WorkID, createdWorks[0].Title)

	retryCode, retryMsg, retryWorkID := mustSubmitWork(
		t,
		client,
		submissionBaseURL,
		limitSession,
		limitTrackID,
		"retry after delete",
		map[string]any{"scenario": "S5_retry_after_delete"},
	)
	if retryCode != 200 {
		t.Fatalf("S5 submission after delete should pass, code=%d msg=%s", retryCode, retryMsg)
	}
	mustUploadAndVerify(t, client, submissionBaseURL, env.repoRoot, limitSession, retryWorkID, limitTrackID, supported[4])
	report.ScenarioSummary = append(report.ScenarioSummary, scenarioSummary{
		Name:   "S4_limit_and_S5_delete_then_retry",
		Passed: true,
		Detail: "4th submission blocked, delete one work then submission succeeds",
	})

	// S6: 鐠恒劏绂岄柆鎾剁柈鐠佲€虫倱娑撯偓濮ｆ棁绂屾稉濠囨
	crossContestID, crossTracks := mustCreateContestAndTracks(
		t,
		client,
		adminBaseURL,
		adminToken,
		"e2e_cross_track_limit_"+runID,
		time.Now().Add(-24*time.Hour),
		time.Now().Add(7*24*time.Hour),
	)
	report.ContestIDs["S6_cross_track_limit"] = crossContestID
	mustInstallSubmissionHooks(t, client, adminBaseURL, adminToken, env.repoRoot, runID+"_S6", []int{crossTracks[categoryMidShort], crossTracks[categoryScript]})

	crossSession := mustEnsureAuthorSession(t, client, submissionBaseURL, map[string]*authorSession{}, "cross_track_author", runID+"_S6")
	mustExpectSubmissionCode(t, client, submissionBaseURL, crossSession, crossTracks[categoryMidShort], "cross 1", 200)
	mustExpectSubmissionCode(t, client, submissionBaseURL, crossSession, crossTracks[categoryMidShort], "cross 2", 200)
	mustExpectSubmissionCode(t, client, submissionBaseURL, crossSession, crossTracks[categoryScript], "cross 3", 200)
	code4, msg4, _ := mustSubmitWork(t, client, submissionBaseURL, crossSession, crossTracks[categoryScript], "cross 4", map[string]any{"scenario": "S6"})
	crossPass := code4 == 500 && strings.Contains(strings.ToLower(msg4), "at most 3")
	report.ScenarioSummary = append(report.ScenarioSummary, scenarioSummary{
		Name:   "S6_cross_track_limit",
		Passed: crossPass,
		Detail: fmt.Sprintf("4th cross-track submission code=%d msg=%s", code4, msg4),
	})

	// S7: file_post 閼存碍婀版潏鎾冲毉闂堢偞纭禞SON
	invalidContestID, invalidTracks := mustCreateContestAndTracks(
		t,
		client,
		adminBaseURL,
		adminToken,
		"e2e_invalid_script_"+runID,
		time.Now().Add(-24*time.Hour),
		time.Now().Add(7*24*time.Hour),
	)
	report.ContestIDs["S7_invalid_script_output"] = invalidContestID
	invalidTrackID := invalidTracks[categoryUltra]
	mustInstallInvalidFilePostHook(t, client, adminBaseURL, adminToken, runID+"_S7", invalidTrackID)

	invalidSession := mustEnsureAuthorSession(t, client, submissionBaseURL, map[string]*authorSession{}, "invalid_script_author", runID+"_S7")
	invCode, invMsg, invWorkID := mustSubmitWork(
		t,
		client,
		submissionBaseURL,
		invalidSession,
		invalidTrackID,
		"invalid script output",
		map[string]any{"scenario": "S7"},
	)
	if invCode != 200 {
		t.Fatalf("S7 submission should pass before upload, code=%d msg=%s", invCode, invMsg)
	}
	invFile := pickSupportedDocx(files, 1)
	if len(invFile) == 0 {
		t.Fatal("no supported .docx file found for S7")
	}
	uploadCode, uploadMsg := mustUploadFile(t, client, submissionBaseURL, invalidSession.Token, invWorkID, invFile[0].AbsPath)
	invalidPass := uploadCode == 500 && strings.Contains(strings.ToLower(uploadMsg), "invalid script output json")
	report.ScenarioSummary = append(report.ScenarioSummary, scenarioSummary{
		Name:   "S7_invalid_script_output",
		Passed: invalidPass,
		Detail: fmt.Sprintf("upload code=%d msg=%s", uploadCode, uploadMsg),
	})
	// S8: fallback filename parsing
	fallbackFile, hasFallback := pickFallbackDocx(files)
	if hasFallback {
		fallbackContestID, fallbackTracks := mustCreateContestAndTracks(
			t,
			client,
			adminBaseURL,
			adminToken,
			"e2e_fallback_name_"+runID,
			time.Now().Add(-24*time.Hour),
			time.Now().Add(7*24*time.Hour),
		)
		report.ContestIDs["S8_fallback_parse"] = fallbackContestID
		fallbackTrackID := fallbackTracks[categoryMidShort]
		mustInstallSubmissionHooks(t, client, adminBaseURL, adminToken, env.repoRoot, runID+"_S8", []int{fallbackTrackID})
		fallbackSession := mustEnsureAuthorSession(t, client, submissionBaseURL, map[string]*authorSession{}, fallbackFile.AuthorIdentity, runID+"_S8")
		code, msg, workID := mustSubmitWork(t, client, submissionBaseURL, fallbackSession, fallbackTrackID, fallbackFile.TitleRaw, map[string]any{"scenario": "S8"})
		if code != 200 {
			t.Fatalf("S8 submission should pass, code=%d msg=%s", code, msg)
		}
		upCode, upMsg := mustUploadFile(t, client, submissionBaseURL, fallbackSession.Token, workID, fallbackFile.AbsPath)
		fallbackPass := upCode == 200
		report.ScenarioSummary = append(report.ScenarioSummary, scenarioSummary{
			Name:   "S8_fallback_filename_parse",
			Passed: fallbackPass,
			Detail: fmt.Sprintf("upload code=%d msg=%s", upCode, upMsg),
		})
	} else {
		report.ScenarioSummary = append(report.ScenarioSummary, scenarioSummary{
			Name:   "S8_fallback_filename_parse",
			Passed: true,
			Detail: "no fallback docx file found, scenario skipped",
		})
	}

	reportPath := mustWriteReport(t, env.repoRoot, report)
	t.Logf("full chain report written: %s", reportPath)

	if len(report.FileRunResults) != len(files) {
		t.Fatalf("S1 full-file execution mismatch, expected %d got %d", len(files), len(report.FileRunResults))
	}

	for _, summary := range report.ScenarioSummary {
		if !summary.Passed {
			t.Fatalf("scenario failed: %s detail=%s", summary.Name, summary.Detail)
		}
	}
}

func mustSetupStrongEnv(t *testing.T) strongEnv {
	t.Helper()

	strongSetupOnce.Do(func() {
		repoRoot, err := findRepoRoot(".")
		if err != nil {
			strongSetupErr = err
			return
		}

		if err = os.Chdir(repoRoot); err != nil {
			strongSetupErr = err
			return
		}

		cfg, err := conf.ReadConfig()
		if err != nil {
			strongSetupErr = err
			return
		}

		if strings.TrimSpace(os.Getenv("Ubik_JWT_Access_Key")) == "" {
			_ = os.Setenv("Ubik_JWT_Access_Key", "strong-e2e-access-key-2026")
		}
		if strings.TrimSpace(os.Getenv("Ubik_JWT_Refresh_Key")) == "" {
			_ = os.Setenv("Ubik_JWT_Refresh_Key", "strong-e2e-refresh-key-2026")
		}

		log.InitLoggr(cfg.Log)
		if err = token.InitJWT(cfg.System.Token); err != nil {
			strongSetupErr = err
			return
		}
		if err = pgsql.Start(cfg.DB); err != nil {
			strongSetupErr = err
			return
		}
		if err = redis.InitRedis(cfg.Redis); err != nil {
			strongSetupErr = err
			return
		}

		strongSetupEnv = strongEnv{repoRoot: repoRoot, config: cfg}
	})

	if strongSetupErr != nil {
		t.Fatalf("strong dependency setup failed: %v", strongSetupErr)
	}

	return strongSetupEnv
}

func findRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		dir = parent
	}
}

func mustAdminLogin(t *testing.T, client *http.Client, adminBaseURL string, adminPassword string) string {
	t.Helper()

	adminInfo, err := pgsql.FindAdminByUsername("superadmin")
	if err != nil {
		t.Fatalf("find superadmin failed: %v", err)
	}
	hash, err := password.HashPassword(adminPassword)
	if err != nil {
		t.Fatalf("hash admin password failed: %v", err)
	}
	if err = pgsql.ChangeAdminPassword(adminInfo.AdminID, hash); err != nil {
		t.Fatalf("set admin password failed: %v", err)
	}

	resp := mustDoJSON(
		t,
		client,
		http.MethodPost,
		adminBaseURL+"/api/v1/admin/login",
		"",
		map[string]any{"adminName": "superadmin", "password": adminPassword},
	)
	if resp.Code != 200 {
		t.Fatalf("admin login failed: code=%d body=%s", resp.Code, resp.Body)
	}
	msgMap := mustMap(t, resp.Msg)
	tokenValue, ok := msgMap["access_token"].(string)
	if !ok || strings.TrimSpace(tokenValue) == "" {
		t.Fatalf("admin login token missing: %+v", msgMap)
	}

	return "Bearer " + tokenValue
}

func mustCreateContestAndTracks(t *testing.T, client *http.Client, adminBaseURL string, adminToken string, prefix string, start time.Time, end time.Time) (int, map[string]int) {
	t.Helper()

	contestResp := mustDoJSON(
		t,
		client,
		http.MethodPost,
		adminBaseURL+"/api/v1/admin/contest",
		adminToken,
		map[string]any{
			"contestName":         prefix,
			"contestStartDate":    start.UTC().Format(time.RFC3339),
			"contestEndDate":      end.UTC().Format(time.RFC3339),
			"contestIntroduction": "strong dependency full chain test",
		},
	)
	if contestResp.Code != 200 {
		t.Fatalf("create contest failed: code=%d body=%s", contestResp.Code, contestResp.Body)
	}
	contestMsg := mustMap(t, contestResp.Msg)
	contestID := mustInt(t, contestMsg["contestID"])

	trackByCategory := map[string]int{}
	categories := []string{categoryMidShort, categoryScript, categoryUltra}
	for _, category := range categories {
		trackResp := mustDoJSON(
			t,
			client,
			http.MethodPost,
			adminBaseURL+"/api/v1/admin/track",
			adminToken,
			map[string]any{
				"trackName":        prefix + "_" + category,
				"contestID":        contestID,
				"trackDescription": "auto generated for strong dependency test",
				"trackSettings": map[string]any{
					"sourceCategory": category,
				},
			},
		)
		if trackResp.Code != 200 {
			t.Fatalf("create track failed category=%s code=%d body=%s", category, trackResp.Code, trackResp.Body)
		}
		trackMsg := mustMap(t, trackResp.Msg)
		trackByCategory[category] = mustInt(t, trackMsg["trackID"])
	}

	return contestID, trackByCategory
}

func mustInstallSubmissionHooks(t *testing.T, client *http.Client, adminBaseURL string, adminToken string, repoRoot string, keySuffix string, trackIDs []int) {
	t.Helper()

	safeSuffix := sanitizeKey(keySuffix)
	if safeSuffix == "" {
		safeSuffix = time.Now().Format("150405")
	}

	limitScriptKey := "limit_three_submission_" + safeSuffix
	wordScriptKey := "count_docx_words_" + safeSuffix

	limitScriptID := mustCreateScriptDefinition(t, client, adminBaseURL, adminToken, limitScriptKey, "Limit Three", "python")
	limitVersionID := mustUploadScriptVersion(t, client, adminBaseURL, adminToken, limitScriptID, filepath.Join(repoRoot, "scripts", "submission_hooks", "limit_three_submissions.py"))

	wordScriptID := mustCreateScriptDefinition(t, client, adminBaseURL, adminToken, wordScriptKey, "Count Words", "python")
	wordVersionID := mustUploadScriptVersion(t, client, adminBaseURL, adminToken, wordScriptID, filepath.Join(repoRoot, "scripts", "word_num_count", "v1", "count_docx_words.py"))

	limitFlowID := mustCreateScriptFlow(t, client, adminBaseURL, adminToken, "submission_pre_limit_"+safeSuffix, "Submission Pre Limit")
	mustReplaceFlowSteps(t, client, adminBaseURL, adminToken, limitFlowID, []map[string]any{{
		"stepName":        "limit-three-submissions",
		"stepOrder":       1,
		"scriptID":        limitScriptID,
		"scriptVersionID": limitVersionID,
		"timeoutMs":       10000,
		"failureStrategy": "fail_close",
		"inputTemplate": map[string]any{
			"maxCount": 3,
		},
	}})

	wordFlowID := mustCreateScriptFlow(t, client, adminBaseURL, adminToken, "file_post_word_count_"+safeSuffix, "File Post Word Count")
	mustReplaceFlowSteps(t, client, adminBaseURL, adminToken, wordFlowID, []map[string]any{{
		"stepName":        "count-docx-words",
		"stepOrder":       1,
		"scriptID":        wordScriptID,
		"scriptVersionID": wordVersionID,
		"timeoutMs":       20000,
		"failureStrategy": "fail_close",
	}})

	for _, trackID := range trackIDs {
		mustCreateFlowMount(t, client, adminBaseURL, adminToken, limitFlowID, "submission", "submission_pre", "track", trackID)
		mustCreateFlowMount(t, client, adminBaseURL, adminToken, wordFlowID, "submission", "file_post", "track", trackID)
	}
}

func mustInstallInvalidFilePostHook(t *testing.T, client *http.Client, adminBaseURL string, adminToken string, keySuffix string, trackID int) {
	t.Helper()

	tmpScript, err := os.CreateTemp("", "invalid_script_*.py")
	if err != nil {
		t.Fatalf("create temp script failed: %v", err)
	}
	scriptPath := tmpScript.Name()
	_, _ = tmpScript.WriteString("#!/usr/bin/env python3\nprint('NOT_JSON_OUTPUT')\n")
	_ = tmpScript.Close()
	t.Cleanup(func() { _ = os.Remove(scriptPath) })

	safeSuffix := sanitizeKey(keySuffix)
	if safeSuffix == "" {
		safeSuffix = time.Now().Format("150405")
	}

	scriptID := mustCreateScriptDefinition(t, client, adminBaseURL, adminToken, "invalid_output_"+safeSuffix, "Invalid Output Script", "python")
	versionID := mustUploadScriptVersion(t, client, adminBaseURL, adminToken, scriptID, scriptPath)
	flowID := mustCreateScriptFlow(t, client, adminBaseURL, adminToken, "invalid_output_flow_"+safeSuffix, "Invalid Output Flow")
	mustReplaceFlowSteps(t, client, adminBaseURL, adminToken, flowID, []map[string]any{{
		"stepName":        "invalid-output",
		"stepOrder":       1,
		"scriptID":        scriptID,
		"scriptVersionID": versionID,
		"timeoutMs":       5000,
		"failureStrategy": "fail_close",
	}})
	mustCreateFlowMount(t, client, adminBaseURL, adminToken, flowID, "submission", "file_post", "track", trackID)
}

func mustCreateScriptDefinition(t *testing.T, client *http.Client, adminBaseURL string, adminToken string, scriptKey string, scriptName string, interpreter string) int {
	t.Helper()

	resp := mustDoJSON(
		t,
		client,
		http.MethodPost,
		adminBaseURL+"/api/v1/admin/scripts",
		adminToken,
		map[string]any{
			"scriptKey":   scriptKey,
			"scriptName":  scriptName,
			"interpreter": interpreter,
			"description": "strong dependency e2e",
			"isEnabled":   true,
			"meta":        map[string]any{"scope": "submission"},
		},
	)
	if resp.Code != 200 {
		t.Fatalf("create script definition failed: code=%d body=%s", resp.Code, resp.Body)
	}
	msgMap := mustMap(t, resp.Msg)
	return mustInt(t, msgMap["scriptID"])
}

func mustUploadScriptVersion(t *testing.T, client *http.Client, adminBaseURL string, adminToken string, scriptID int, localFilePath string) int {
	t.Helper()

	content, err := os.ReadFile(localFilePath)
	if err != nil {
		t.Fatalf("read script file failed: %v", err)
	}

	resp := mustDoMultipart(
		t,
		client,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/admin/scripts/%d/versions/upload", adminBaseURL, scriptID),
		adminToken,
		map[string]string{},
		"script_file",
		filepath.Base(localFilePath),
		content,
	)
	if resp.Code != 200 {
		t.Fatalf("upload script version failed: code=%d body=%s", resp.Code, resp.Body)
	}
	msgMap := mustMap(t, resp.Msg)
	return mustInt(t, msgMap["versionID"])
}

func mustCreateScriptFlow(t *testing.T, client *http.Client, adminBaseURL string, adminToken string, flowKey string, flowName string) int {
	t.Helper()

	resp := mustDoJSON(
		t,
		client,
		http.MethodPost,
		adminBaseURL+"/api/v1/admin/script-flows",
		adminToken,
		map[string]any{
			"flowKey":     flowKey,
			"flowName":    flowName,
			"description": "strong dependency e2e",
			"isEnabled":   true,
		},
	)
	if resp.Code != 200 {
		t.Fatalf("create script flow failed: code=%d body=%s", resp.Code, resp.Body)
	}
	msgMap := mustMap(t, resp.Msg)
	return mustInt(t, msgMap["flowID"])
}

func mustReplaceFlowSteps(t *testing.T, client *http.Client, adminBaseURL string, adminToken string, flowID int, steps []map[string]any) {
	t.Helper()

	resp := mustDoJSON(
		t,
		client,
		http.MethodPut,
		fmt.Sprintf("%s/api/v1/admin/script-flows/%d/steps", adminBaseURL, flowID),
		adminToken,
		steps,
	)
	if resp.Code != 200 {
		t.Fatalf("replace flow steps failed: code=%d body=%s", resp.Code, resp.Body)
	}
}

func mustCreateFlowMount(t *testing.T, client *http.Client, adminBaseURL string, adminToken string, flowID int, scope string, eventKey string, targetType string, targetID int) {
	t.Helper()

	resp := mustDoJSON(
		t,
		client,
		http.MethodPost,
		adminBaseURL+"/api/v1/admin/script-flows/mounts",
		adminToken,
		map[string]any{
			"flowID":     flowID,
			"scope":      scope,
			"eventKey":   eventKey,
			"targetType": targetType,
			"targetID":   targetID,
			"isEnabled":  true,
		},
	)
	if resp.Code != 200 {
		t.Fatalf("create flow mount failed: code=%d body=%s", resp.Code, resp.Body)
	}
}

func mustEnsureAuthorSession(t *testing.T, client *http.Client, submissionBaseURL string, cache map[string]*authorSession, authorIdentity string, suffix string) *authorSession {
	t.Helper()

	if session, ok := cache[authorIdentity]; ok {
		return session
	}

	authorNameBase := normalizeText(authorIdentity)
	if authorNameBase == "" {
		authorNameBase = "unknown_author"
	}
	authorName := truncateString(authorNameBase+"_"+sanitizeKey(suffix), 120)
	passwordValue := "AuthorPass!2026"
	email := fmt.Sprintf("e2e_%s@example.com", shortHash(authorName))

	registerResp := mustDoJSON(
		t,
		client,
		http.MethodPost,
		submissionBaseURL+"/api/v1/author/register",
		"",
		map[string]any{
			"authorName":  authorName,
			"password":    passwordValue,
			"authorEmail": email,
			"penName":     authorIdentity,
		},
	)
	if registerResp.Code != 200 {
		t.Fatalf("author register failed for %s: code=%d body=%s", authorName, registerResp.Code, registerResp.Body)
	}

	author := &model.Author{AuthorName: authorName}
	if err := pgsql.GetAuthorByAuthorName(author); err != nil {
		t.Fatalf("query author after register failed: %v", err)
	}

	loginResp := mustDoJSON(
		t,
		client,
		http.MethodPost,
		submissionBaseURL+"/api/v1/author/login",
		"",
		map[string]any{
			"authorID": author.AuthorID,
			"password": passwordValue,
		},
	)
	if loginResp.Code != 200 {
		t.Fatalf("author login failed for %s: code=%d body=%s", authorName, loginResp.Code, loginResp.Body)
	}
	msgMap := mustMap(t, loginResp.Msg)
	accessToken, ok := msgMap["access_token"].(string)
	if !ok || strings.TrimSpace(accessToken) == "" {
		t.Fatalf("author login token missing for %s: %+v", authorName, msgMap)
	}

	session := &authorSession{
		AuthorName: authorName,
		AuthorID:   author.AuthorID,
		Token:      "Bearer " + accessToken,
	}
	cache[authorIdentity] = session
	return session
}

func runOneFileThroughSubmission(
	t *testing.T,
	client *http.Client,
	submissionBaseURL string,
	repoRoot string,
	scenario string,
	file testInputFile,
	session *authorSession,
	trackID int,
) fileRunResult {
	t.Helper()

	result := fileRunResult{
		Scenario:      scenario,
		FilePath:      file.RelPath,
		Category:      file.Category,
		Extension:     file.Ext,
		AuthorRaw:     file.AuthorRaw,
		TitleRaw:      file.TitleRaw,
		ParseFallback: file.ParseFallback,
		TrackID:       trackID,
		AuthorID:      session.AuthorID,
	}

	code, msg, workID := mustSubmitWork(
		t,
		client,
		submissionBaseURL,
		session,
		trackID,
		file.TitleRaw,
		map[string]any{
			"source_file": file.RelPath,
			"source_ext":  file.Ext,
			"scenario":    scenario,
		},
	)
	result.SubmissionCode = code
	result.SubmissionMsg = msg
	result.WorkID = workID

	if code != 200 {
		return result
	}

	uploadCode, uploadMsg := mustUploadFile(t, client, submissionBaseURL, session.Token, workID, file.AbsPath)
	result.UploadCode = uploadCode
	result.UploadMsg = uploadMsg

	if uploadCode == 200 {
		savedPath := filepath.Join(repoRoot, "files", "submissions", strconv.Itoa(trackID), strconv.Itoa(session.AuthorID), strconv.Itoa(workID)+".docx")
		result.SavedDocx = filepath.ToSlash(savedPath)
		if _, err := os.Stat(savedPath); err != nil {
			result.UploadMsg = "saved file missing: " + err.Error()
		}
		work := model.Work{WorkID: workID}
		if err := pgsql.GetSubmissionByWorkID(&work); err == nil {
			result.WordCount = parseWordCount(work.WorkInfos)
		}
	}

	return result
}

func mustSubmitWork(t *testing.T, client *http.Client, submissionBaseURL string, session *authorSession, trackID int, title string, infos map[string]any) (int, string, int) {
	t.Helper()

	resp := mustDoJSON(
		t,
		client,
		http.MethodPost,
		submissionBaseURL+"/api/v1/author/submission",
		session.Token,
		map[string]any{
			"authorID":  session.AuthorID,
			"trackID":   trackID,
			"workTitle": normalizeText(title),
			"workInfos": infos,
		},
	)

	msgText := messageToString(resp.Msg)
	workID := 0
	if resp.Code == 200 {
		msgMap := mustMap(t, resp.Msg)
		workID = mustInt(t, msgMap["workID"])
	}
	return resp.Code, msgText, workID
}

func mustUploadFile(t *testing.T, client *http.Client, submissionBaseURL string, tokenValue string, workID int, filePath string) (int, string) {
	t.Helper()

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read upload file failed: %v", err)
	}
	resp := mustDoMultipart(
		t,
		client,
		http.MethodPost,
		submissionBaseURL+"/api/v1/author/submission/file",
		tokenValue,
		map[string]string{"work_id": strconv.Itoa(workID)},
		"article_file",
		filepath.Base(filePath),
		content,
	)
	return resp.Code, messageToString(resp.Msg)
}

func mustDeleteWork(t *testing.T, client *http.Client, submissionBaseURL string, session *authorSession, trackID int, workID int, title string) {
	t.Helper()

	resp := mustDoJSON(
		t,
		client,
		http.MethodDelete,
		submissionBaseURL+"/api/v1/author/submission",
		session.Token,
		map[string]any{
			"workID":    workID,
			"authorID":  session.AuthorID,
			"trackID":   trackID,
			"workTitle": title,
		},
	)
	if resp.Code != 200 {
		t.Fatalf("delete work failed: code=%d body=%s", resp.Code, resp.Body)
	}
}

func mustUploadAndVerify(t *testing.T, client *http.Client, submissionBaseURL string, repoRoot string, session *authorSession, workID int, trackID int, file testInputFile) {
	t.Helper()

	code, msg := mustUploadFile(t, client, submissionBaseURL, session.Token, workID, file.AbsPath)
	if code != 200 {
		t.Fatalf("upload should pass for file=%s code=%d msg=%s", file.RelPath, code, msg)
	}
	savedPath := filepath.Join(repoRoot, "files", "submissions", strconv.Itoa(trackID), strconv.Itoa(session.AuthorID), strconv.Itoa(workID)+".docx")
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
}

func mustExpectSubmissionCode(t *testing.T, client *http.Client, submissionBaseURL string, session *authorSession, trackID int, title string, expected int) {
	t.Helper()

	code, msg, _ := mustSubmitWork(t, client, submissionBaseURL, session, trackID, title, map[string]any{"scenario": "expect_code"})
	if code != expected {
		t.Fatalf("unexpected submission code title=%s expected=%d got=%d msg=%s", title, expected, code, msg)
	}
}

func mustDoJSON(t *testing.T, client *http.Client, method string, url string, tokenValue string, payload any) apiCallResult {
	t.Helper()

	var bodyReader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal json payload failed: %v", err)
		}
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if tokenValue != "" {
		req.Header.Set("Authorization", tokenValue)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("http request failed: %v", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body failed: %v", err)
	}

	var envelope map[string]any
	if err = json.Unmarshal(rawBody, &envelope); err != nil {
		t.Fatalf("unmarshal response failed: %v body=%s", err, string(rawBody))
	}

	return apiCallResult{
		Code: mustInt(t, envelope["code"]),
		Msg:  envelope["msg"],
		Body: string(rawBody),
	}
}

func mustDoMultipart(
	t *testing.T,
	client *http.Client,
	method string,
	url string,
	tokenValue string,
	fields map[string]string,
	fileField string,
	fileName string,
	fileContent []byte,
) apiCallResult {
	t.Helper()

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write multipart field failed: %v", err)
		}
	}
	part, err := writer.CreateFormFile(fileField, fileName)
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err = part.Write(fileContent); err != nil {
		t.Fatalf("write multipart file failed: %v", err)
	}
	if err = writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	req, err := http.NewRequest(method, url, &payload)
	if err != nil {
		t.Fatalf("new multipart request failed: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if tokenValue != "" {
		req.Header.Set("Authorization", tokenValue)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("http multipart request failed: %v", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read multipart response body failed: %v", err)
	}

	var envelope map[string]any
	if err = json.Unmarshal(rawBody, &envelope); err != nil {
		t.Fatalf("unmarshal multipart response failed: %v body=%s", err, string(rawBody))
	}

	return apiCallResult{
		Code: mustInt(t, envelope["code"]),
		Msg:  envelope["msg"],
		Body: string(rawBody),
	}
}

func mustScanAllTestFiles(t *testing.T, repoRoot string) []testInputFile {
	t.Helper()

	root := filepath.Join(repoRoot, "tests_files")
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("tests_files not found: %v", err)
	}

	files := make([]testInputFile, 0, 512)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		parts := strings.Split(rel, "/")
		category := "unknown"
		if len(parts) > 1 {
			category = parts[0]
		}

		base := filepath.Base(path)
		ext := strings.ToLower(filepath.Ext(base))
		nameNoExt := strings.TrimSuffix(base, filepath.Ext(base))
		author, title, fallback := parseAuthorAndTitle(nameNoExt)

		files = append(files, testInputFile{
			AbsPath:        path,
			RelPath:        filepath.ToSlash(filepath.Join("tests_files", rel)),
			Category:       category,
			Ext:            ext,
			AuthorRaw:      author,
			TitleRaw:       title,
			ParseFallback:  fallback,
			NameNoExt:      nameNoExt,
			OriginalBase:   base,
			AuthorIdentity: author,
		})

		return nil
	})
	if err != nil {
		t.Fatalf("scan tests_files failed: %v", err)
	}

	sort.Slice(files, func(i int, j int) bool {
		return files[i].RelPath < files[j].RelPath
	})

	return files
}

func parseAuthorAndTitle(nameNoExt string) (string, string, bool) {
	parts := strings.Split(nameNoExt, "_")
	fallback := false
	author := ""
	title := ""

	if len(parts) >= 1 {
		author = normalizeText(parts[0])
	}
	if len(parts) >= 3 {
		title = normalizeText(parts[2])
	} else {
		fallback = true
	}

	if author == "" {
		author = "unknown_author"
		fallback = true
	}
	if title == "" {
		title = normalizeText(nameNoExt)
		if title == "" {
			title = "untitled"
		}
		fallback = true
	}

	return author, title, fallback
}

func pickSupportedDocx(files []testInputFile, n int) []testInputFile {
	selected := make([]testInputFile, 0, n)
	for _, file := range files {
		if file.Ext == ".docx" {
			selected = append(selected, file)
			if len(selected) == n {
				break
			}
		}
	}
	return selected
}

func pickFallbackDocx(files []testInputFile) (testInputFile, bool) {
	for _, file := range files {
		if file.Ext == ".docx" && file.ParseFallback {
			return file, true
		}
	}
	return testInputFile{}, false
}

func mustWriteReport(t *testing.T, repoRoot string, report fullChainReport) string {
	t.Helper()

	reportDir := filepath.Join(repoRoot, "tests_files", "private_e2e_results")
	if err := os.MkdirAll(reportDir, os.ModePerm); err != nil {
		t.Fatalf("create report dir failed: %v", err)
	}

	fileName := "full_chain_strong_" + report.RunID + ".json"
	reportPath := filepath.Join(reportDir, fileName)

	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal report failed: %v", err)
	}
	if err = os.WriteFile(reportPath, content, 0o644); err != nil {
		t.Fatalf("write report failed: %v", err)
	}

	return filepath.ToSlash(reportPath)
}

func parseWordCount(infos map[string]any) int {
	if infos == nil {
		return 0
	}
	value, ok := infos["word_count"]
	if !ok {
		return 0
	}
	return toInt(value)
}

func messageToString(msg any) string {
	switch m := msg.(type) {
	case string:
		return m
	case map[string]any:
		b, _ := json.Marshal(m)
		return string(b)
	case []any:
		b, _ := json.Marshal(m)
		return string(b)
	default:
		b, _ := json.Marshal(m)
		return string(b)
	}
}

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()
	m, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map response, got %T (%v)", value, value)
	}
	return m
}

func mustInt(t *testing.T, value any) int {
	t.Helper()
	i, ok := asInt(value)
	if !ok {
		t.Fatalf("expected int value, got %T (%v)", value, value)
	}
	return i
}

func asInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		i64, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(i64), true
	default:
		return 0, false
	}
}

func toInt(value any) int {
	i, ok := asInt(value)
	if !ok {
		return 0
	}
	return i
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	return strings.Join(strings.Fields(value), " ")
}

func sanitizeKey(value string) string {
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			builder.WriteRune(r)
		}
	}
	result := builder.String()
	result = strings.Trim(result, "_-")
	if len(result) > 32 {
		result = result[:32]
	}
	if result == "" {
		return "k"
	}
	return result
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func truncateString(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxLen {
		return value
	}
	return string(runes[:maxLen])
}
