package submission

import (
	"context"
	"encoding/json"
	"errors"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/scriptflow"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type privateE2EStore struct {
	mu sync.Mutex

	nextAuthorID int
	nextWorkID   int

	authorsByID    map[int]model.Author
	authorIDByName map[string]int
	worksByID      map[int]model.Work
	uploadPerm     map[int]struct {
		authorID int
		trackID  int
	}
}

type privateDocCandidate struct {
	path string
	size int64
}

type privateSubmissionAttempt struct {
	Index    int    `json:"index"`
	FilePath string `json:"filePath"`
	Code     int    `json:"code"`
	WorkID   int    `json:"workID,omitempty"`
	Message  string `json:"message,omitempty"`
}

type privateUploadAttempt struct {
	WorkID      int    `json:"workID"`
	SourceFile  string `json:"sourceFile"`
	Code        int    `json:"code"`
	WordCount   int    `json:"wordCount"`
	SavedDocx   string `json:"savedDocx"`
	UploadError string `json:"uploadError,omitempty"`
}

type privateE2EReport struct {
	GeneratedAt       string                     `json:"generatedAt"`
	Interpreter       string                     `json:"interpreter"`
	TrackID           int                        `json:"trackID"`
	AuthorID          int                        `json:"authorID"`
	SubmissionResults []privateSubmissionAttempt `json:"submissionResults"`
	UploadResults     []privateUploadAttempt     `json:"uploadResults"`
	WorkInfos         map[string]map[string]any  `json:"workInfos"`
}

func newPrivateE2EStore() *privateE2EStore {
	return &privateE2EStore{
		nextAuthorID:   0,
		nextWorkID:     0,
		authorsByID:    map[int]model.Author{},
		authorIDByName: map[string]int{},
		worksByID:      map[int]model.Work{},
		uploadPerm: map[int]struct {
			authorID int
			trackID  int
		}{},
	}
}

func (s *privateE2EStore) getAuthorByName(author *model.Author) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.authorIDByName[author.AuthorName]
	if !ok {
		return _const.UsernameNotExist
	}

	stored := s.authorsByID[id]
	*author = stored
	return nil
}

func (s *privateE2EStore) createAuthor(author *model.Author) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.authorIDByName[author.AuthorName]; exists {
		return errors.New("username already exists")
	}

	s.nextAuthorID++
	created := *author
	created.AuthorID = s.nextAuthorID

	s.authorsByID[created.AuthorID] = created
	s.authorIDByName[created.AuthorName] = created.AuthorID
	*author = created

	return nil
}

func (s *privateE2EStore) getAuthorByID(author *model.Author) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.authorsByID[author.AuthorID]
	if !ok {
		return errors.New("author not found")
	}

	*author = stored
	return nil
}

func (s *privateE2EStore) updateAuthor(author *model.Author) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.authorsByID[author.AuthorID]
	if !ok {
		return errors.New("author not found")
	}

	stored.PenName = author.PenName
	stored.AuthorEmail = author.AuthorEmail
	stored.AuthorInfos = author.AuthorInfos
	s.authorsByID[author.AuthorID] = stored
	return nil
}

func (s *privateE2EStore) createWork(work *model.Work) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextWorkID++
	created := *work
	created.WorkID = s.nextWorkID
	if created.WorkInfos == nil {
		created.WorkInfos = map[string]any{}
	}

	s.worksByID[created.WorkID] = created
	*work = created
	return nil
}

func (s *privateE2EStore) updateWork(work *model.Work) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.worksByID[work.WorkID]; !ok {
		return errors.New("work not found")
	}

	s.worksByID[work.WorkID] = *work
	return nil
}

func (s *privateE2EStore) deleteWork(work *model.Work) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.worksByID, work.WorkID)
	delete(s.uploadPerm, work.WorkID)
	return nil
}

func (s *privateE2EStore) countWorksByAuthorAndTrack(authorID int, trackID int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int64
	for _, work := range s.worksByID {
		if work.AuthorID == authorID && work.TrackID == trackID {
			count++
		}
	}

	return count, nil
}

func (s *privateE2EStore) countWorksByAuthorAndContest(authorID int, contestID int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int64
	for _, work := range s.worksByID {
		if work.AuthorID == authorID {
			count++
		}
	}

	return count, nil
}

func (s *privateE2EStore) listWorksByAuthorID(authorID int) ([]model.Work, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	works := make([]model.Work, 0)
	for _, work := range s.worksByID {
		if work.AuthorID == authorID {
			works = append(works, work)
		}
	}

	sort.Slice(works, func(i int, j int) bool {
		return works[i].WorkID < works[j].WorkID
	})

	return works, nil
}

func (s *privateE2EStore) setUploadPermission(authorID int, trackID int, workID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.uploadPerm[workID] = struct {
		authorID int
		trackID  int
	}{
		authorID: authorID,
		trackID:  trackID,
	}
	return nil
}

func (s *privateE2EStore) getUploadPermission(workID int) (int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	permission, ok := s.uploadPerm[workID]
	if !ok {
		return 0, 0, errors.New("upload permission not found")
	}

	return permission.authorID, permission.trackID, nil
}

func (s *privateE2EStore) patchWorkInfos(workID int, patch map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	work, ok := s.worksByID[workID]
	if !ok {
		return errors.New("work not found")
	}

	if work.WorkInfos == nil {
		work.WorkInfos = map[string]any{}
	}
	for key, value := range patch {
		work.WorkInfos[key] = value
	}

	s.worksByID[workID] = work
	return nil
}

func (s *privateE2EStore) getWork(workID int) (model.Work, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	work, ok := s.worksByID[workID]
	return work, ok
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
			return "", errors.New("go.mod not found in parent directories")
		}
		dir = parent
	}
}

func collectPrivateDocxFiles(repoRoot string, minCount int) ([]privateDocCandidate, error) {
	root := filepath.Join(repoRoot, "tests_files")
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}

	candidates := make([]privateDocCandidate, 0)
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".docx") {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}

		candidates = append(candidates, privateDocCandidate{path: path, size: info.Size()})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	if len(candidates) < minCount {
		return nil, errors.New("not enough docx files in tests_files")
	}

	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].size == candidates[j].size {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].size < candidates[j].size
	})

	return candidates[:minCount], nil
}

func decodeRespBodyMap(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v body=%s", err, string(body))
	}

	return resp
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

func toRelativeSlashPath(repoRoot string, absolutePath string) string {
	rel, err := filepath.Rel(repoRoot, absolutePath)
	if err != nil {
		return filepath.ToSlash(absolutePath)
	}
	return filepath.ToSlash(rel)
}

func TestPrivateFilesE2ERegisterToUploadWithHooks(t *testing.T) {
	backupSubmissionHooks(t)

	repoRoot, err := findRepoRoot(".")
	if err != nil {
		t.Fatalf("find repo root failed: %v", err)
	}

	selectedFiles, err := collectPrivateDocxFiles(repoRoot, 4)
	if err != nil {
		t.Skipf("private docx files not available, skip private e2e: %v", err)
	}

	oldWD, _ := os.Getwd()
	if chdirErr := os.Chdir(repoRoot); chdirErr != nil {
		t.Fatalf("chdir repo root failed: %v", chdirErr)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	interpreter := "python"
	if _, lookErr := exec.LookPath(interpreter); lookErr != nil {
		if _, lookPy3Err := exec.LookPath("python3"); lookPy3Err == nil {
			interpreter = "python3"
		} else {
			t.Skip("python interpreter not found")
		}
	}

	const trackID = 2001

	limitScriptPath := filepath.ToSlash(filepath.Join("scripts", "submission_hooks", "limit_three_submissions.py"))
	countScriptPath := filepath.ToSlash(filepath.Join("scripts", "submission_hooks", "count_docx_words.py"))

	store := newPrivateE2EStore()

	getAuthorByAuthorNameFn = store.getAuthorByName
	createAuthorFn = store.createAuthor
	getAuthorByAuthorIDFn = store.getAuthorByID
	updateAuthorFn = store.updateAuthor

	submissionWorkFn = store.createWork
	updateWorkFn = store.updateWork
	deleteWorkFn = store.deleteWork
	findWorksByAuthorIDFn = store.listWorksByAuthorID
	countWorksByAuthorAndTrackFn = store.countWorksByAuthorAndTrack
	countWorksByAuthorAndContestFn = store.countWorksByAuthorAndContest
	getTrackByIDFn = func(trackID int) (model.Track, error) {
		return model.Track{TrackID: trackID, ContestID: 1}, nil
	}

	setUploadFilePermissionFn = store.setUploadPermission
	getUploadFilePermissionFn = store.getUploadPermission
	patchWorkInfosFn = store.patchWorkInfos

	getStartAndEndDateFn = func(_ int) (int64, int64, error) {
		now := time.Now().Unix()
		return now - 3600, now + 3600, nil
	}

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		if scope != scriptflow.ScopeSubmission || targetType != "track" || targetID != trackID {
			return model.ScriptFlow{}, nil, pgsql.ErrFlowNotMounted
		}

		switch eventKey {
		case scriptflow.EventSubmissionPre:
			return model.ScriptFlow{FlowID: 1, FlowKey: "limit_three_submissions", FlowName: "Limit Submission Count", IsEnabled: true},
				[]pgsql.ResolvedFlowStep{{
					Step: model.FlowStep{
						StepID:          1,
						FlowID:          1,
						StepOrder:       1,
						StepName:        "limit-three-submissions",
						ScriptID:        1,
						ScriptVersionID: 1,
						TimeoutMs:       10000,
						FailureStrategy: "fail_close",
						InputTemplate:   map[string]any{"maxCount": 3},
						IsEnabled:       true,
					},
					Script: model.ScriptDefinition{
						ScriptID:    1,
						ScriptKey:   "limit_three_submissions",
						ScriptName:  "Limit Three Submissions",
						Interpreter: interpreter,
						IsEnabled:   true,
						Description: "Block submission when existingCount >= 3",
						Meta:        map[string]any{"type": "private_e2e"},
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
					Version: model.ScriptVersion{
						VersionID:    1,
						ScriptID:     1,
						VersionNum:   1,
						FileName:     filepath.Base(limitScriptPath),
						RelativePath: limitScriptPath,
						IsActive:     true,
						CreatedAt:    time.Now(),
					},
				}}, nil
		case scriptflow.EventFilePost:
			return model.ScriptFlow{FlowID: 2, FlowKey: "count_docx_words", FlowName: "Count Docx Words", IsEnabled: true},
				[]pgsql.ResolvedFlowStep{{
					Step: model.FlowStep{
						StepID:          2,
						FlowID:          2,
						StepOrder:       1,
						StepName:        "count-docx-words",
						ScriptID:        2,
						ScriptVersionID: 2,
						TimeoutMs:       15000,
						FailureStrategy: "fail_close",
						IsEnabled:       true,
					},
					Script: model.ScriptDefinition{
						ScriptID:    2,
						ScriptKey:   "count_docx_words",
						ScriptName:  "Count Docx Words",
						Interpreter: interpreter,
						IsEnabled:   true,
						Description: "Extract and count words from saved docx",
						Meta:        map[string]any{"type": "private_e2e"},
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					},
					Version: model.ScriptVersion{
						VersionID:    2,
						ScriptID:     2,
						VersionNum:   1,
						FileName:     filepath.Base(countScriptPath),
						RelativePath: countScriptPath,
						IsActive:     true,
						CreatedAt:    time.Now(),
					},
				}}, nil
		default:
			return model.ScriptFlow{}, nil, pgsql.ErrFlowNotMounted
		}
	}

	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		executor := scriptflow.NewExecutor(repoRoot, 20*time.Second, []string{"python3", "python"})
		return executor.ExecuteChain(context.Background(), chain, input)
	}

	router := buildSubmissionRouter()

	authorName := "private_e2e_author"
	authorPassword := "S3cret!pass"
	authorID := 1

	registerBody, err := json.Marshal(map[string]any{
		"authorName":  authorName,
		"password":    authorPassword,
		"authorEmail": "private-e2e@example.com",
	})
	if err != nil {
		t.Fatalf("marshal register body failed: %v", err)
	}

	registerResp := doJSONRequest(router, http.MethodPost, "/api/v1/author/register", "", registerBody)
	if code := decodeRespCode(t, registerResp.Body.Bytes()); code != 200 {
		t.Fatalf("register failed: code=%d body=%s", code, registerResp.Body.String())
	}

	loginBody, err := json.Marshal(map[string]any{
		"authorName": authorName,
		"password":   authorPassword,
	})
	if err != nil {
		t.Fatalf("marshal login body failed: %v", err)
	}

	loginResp := doJSONRequest(router, http.MethodPost, "/api/v1/author/login", "", loginBody)
	if code := decodeRespCode(t, loginResp.Body.Bytes()); code != 200 {
		t.Fatalf("login failed: code=%d body=%s", code, loginResp.Body.String())
	}

	loginMap := decodeRespBodyMap(t, loginResp.Body.Bytes())
	loginMsg, ok := loginMap["msg"].(map[string]any)
	if !ok {
		t.Fatalf("invalid login response: %v", loginMap)
	}
	accessToken, ok := loginMsg["access_token"].(string)
	if !ok || strings.TrimSpace(accessToken) == "" {
		t.Fatalf("access token missing in login response: %+v", loginMsg)
	}
	bearerToken := "Bearer " + accessToken

	submissionResults := make([]privateSubmissionAttempt, 0, 4)
	acceptedWorkIDs := make([]int, 0, 3)

	for i := 0; i < 4; i++ {
		candidate := selectedFiles[i]
		title := strings.TrimSpace(strings.TrimSuffix(filepath.Base(candidate.path), filepath.Ext(candidate.path)))
		if title == "" {
			title = "private-e2e-title-" + strconv.Itoa(i+1)
		}

		payload, marshalErr := json.Marshal(map[string]any{
			"authorID":  authorID,
			"trackID":   trackID,
			"workTitle": title,
		})
		if marshalErr != nil {
			t.Fatalf("marshal submission payload failed: %v", marshalErr)
		}

		resp := doJSONRequest(router, http.MethodPost, "/api/v1/author/submission", bearerToken, payload)
		respMap := decodeRespBodyMap(t, resp.Body.Bytes())
		code, _ := toInt(respMap["code"])

		attempt := privateSubmissionAttempt{
			Index:    i + 1,
			FilePath: toRelativeSlashPath(repoRoot, candidate.path),
			Code:     code,
		}

		if i < 3 {
			if code != 200 {
				t.Fatalf("submission #%d should pass, code=%d body=%s", i+1, code, resp.Body.String())
			}
			msgObj, msgOK := respMap["msg"].(map[string]any)
			if !msgOK {
				t.Fatalf("submission #%d response msg should be object, body=%s", i+1, resp.Body.String())
			}
			workID, idOK := toInt(msgObj["workID"])
			if !idOK || workID <= 0 {
				t.Fatalf("submission #%d missing workID, msg=%+v", i+1, msgObj)
			}
			attempt.WorkID = workID
			acceptedWorkIDs = append(acceptedWorkIDs, workID)
		} else {
			if code != 500 {
				t.Fatalf("submission #4 should be blocked by script, code=%d body=%s", code, resp.Body.String())
			}
			if msgText, textOK := respMap["msg"].(string); textOK {
				attempt.Message = msgText
				if !strings.Contains(strings.ToLower(msgText), "at most 3") {
					t.Fatalf("submission #4 blocked message mismatch: %s", msgText)
				}
			}
		}

		submissionResults = append(submissionResults, attempt)
	}

	uploadResults := make([]privateUploadAttempt, 0, len(acceptedWorkIDs))
	workInfos := map[string]map[string]any{}

	for i, workID := range acceptedWorkIDs {
		candidate := selectedFiles[i]
		content, readErr := os.ReadFile(candidate.path)
		if readErr != nil {
			t.Fatalf("read private file failed: %v", readErr)
		}

		resp := doMultipartRequest(
			router,
			"/api/v1/author/submission/file",
			bearerToken,
			map[string]string{"work_id": strconv.Itoa(workID)},
			"article_file",
			filepath.Base(candidate.path),
			content,
		)

		respMap := decodeRespBodyMap(t, resp.Body.Bytes())
		code, _ := toInt(respMap["code"])
		if code != 200 {
			t.Fatalf("upload work_id=%d failed: code=%d body=%s", workID, code, resp.Body.String())
		}

		savedPath := filepath.Join(repoRoot, "submissions", strconv.Itoa(trackID), strconv.Itoa(authorID), strconv.Itoa(workID)+".docx")
		if _, statErr := os.Stat(savedPath); statErr != nil {
			t.Fatalf("saved docx missing for work_id=%d path=%s err=%v", workID, savedPath, statErr)
		}

		work, ok := store.getWork(workID)
		if !ok {
			t.Fatalf("work not found after upload, work_id=%d", workID)
		}

		wordCount, wordOK := toInt(work.WorkInfos["word_count"])
		if !wordOK || wordCount <= 0 {
			t.Fatalf("word_count should be positive for work_id=%d, work_infos=%+v", workID, work.WorkInfos)
		}

		workInfoCopy := map[string]any{}
		for key, value := range work.WorkInfos {
			workInfoCopy[key] = value
		}
		workInfos[strconv.Itoa(workID)] = workInfoCopy

		uploadResults = append(uploadResults, privateUploadAttempt{
			WorkID:     workID,
			SourceFile: toRelativeSlashPath(repoRoot, candidate.path),
			Code:       code,
			WordCount:  wordCount,
			SavedDocx:  toRelativeSlashPath(repoRoot, savedPath),
		})
	}

	report := privateE2EReport{
		GeneratedAt:       time.Now().Format(time.RFC3339),
		Interpreter:       interpreter,
		TrackID:           trackID,
		AuthorID:          authorID,
		SubmissionResults: submissionResults,
		UploadResults:     uploadResults,
		WorkInfos:         workInfos,
	}

	reportDir := filepath.Join(repoRoot, "tests_files", "private_e2e_results")
	if err = os.MkdirAll(reportDir, os.ModePerm); err != nil {
		t.Fatalf("create report dir failed: %v", err)
	}

	reportName := "submission_private_e2e_" + time.Now().Format("20060102_150405") + ".json"
	reportPath := filepath.Join(reportDir, reportName)
	reportBytes, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal report failed: %v", err)
	}
	if err = os.WriteFile(reportPath, reportBytes, 0o644); err != nil {
		t.Fatalf("write report failed: %v", err)
	}

	t.Logf("private e2e report written: %s", toRelativeSlashPath(repoRoot, reportPath))
}
