package pgsql

import (
	"main/conf"
	"main/model"
	_const "main/util/const"
	"testing"
	"time"

	"github.com/glebarez/sqlite"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE admins (admin_id INTEGER PRIMARY KEY AUTOINCREMENT, admin_name TEXT UNIQUE, password TEXT, admin_email TEXT, is_active BOOLEAN);`,
		`CREATE TABLE contests (contest_id INTEGER PRIMARY KEY AUTOINCREMENT, contest_name TEXT, contest_start_date DATE, contest_end_date DATE, contest_introduction TEXT);`,
		`CREATE TABLE tracks (track_id INTEGER PRIMARY KEY AUTOINCREMENT, track_name TEXT, contest_id INTEGER, track_description TEXT, track_settings TEXT);`,
		`CREATE TABLE works (work_id INTEGER PRIMARY KEY AUTOINCREMENT, work_title TEXT, track_id INTEGER, author_id INTEGER, work_infos TEXT);`,
		`CREATE TABLE script_definitions (script_id INTEGER PRIMARY KEY AUTOINCREMENT, script_key TEXT UNIQUE, script_name TEXT, interpreter TEXT, description TEXT, is_enabled BOOLEAN, meta TEXT, created_at DATETIME, updated_at DATETIME);`,
		`CREATE TABLE script_versions (version_id INTEGER PRIMARY KEY AUTOINCREMENT, script_id INTEGER NOT NULL, version_num INTEGER NOT NULL, file_name TEXT NOT NULL, relative_path TEXT NOT NULL, checksum TEXT, is_active BOOLEAN, created_by INTEGER, created_at DATETIME, UNIQUE(script_id, version_num));`,
		`CREATE TABLE script_flows (flow_id INTEGER PRIMARY KEY AUTOINCREMENT, flow_key TEXT UNIQUE, flow_name TEXT, description TEXT, is_enabled BOOLEAN, meta TEXT, created_at DATETIME, updated_at DATETIME);`,
		`CREATE TABLE script_flow_steps (step_id INTEGER PRIMARY KEY AUTOINCREMENT, flow_id INTEGER NOT NULL, step_order INTEGER NOT NULL, step_name TEXT, script_id INTEGER NOT NULL, script_version_id INTEGER, timeout_ms INTEGER, failure_strategy TEXT, input_template TEXT, is_enabled BOOLEAN, UNIQUE(flow_id, step_order));`,
		`CREATE TABLE script_flow_mounts (mount_id INTEGER PRIMARY KEY AUTOINCREMENT, flow_id INTEGER NOT NULL, scope TEXT, event_key TEXT, target_type TEXT, target_id INTEGER, is_enabled BOOLEAN, created_at DATETIME, UNIQUE(scope, event_key, target_type, target_id));`,
		`CREATE TABLE action_logs (log_id INTEGER PRIMARY KEY AUTOINCREMENT, admin_id INTEGER, resource TEXT, action TEXT, created_at DATETIME, details TEXT);`,
		`CREATE TABLE global_config (id INTEGER PRIMARY KEY, is_init BOOLEAN, site_name TEXT, email_address TEXT, email_app_password TEXT, email_smtp_server TEXT, email_smtp_port INTEGER);`,
		`CREATE TABLE author (author_id INTEGER PRIMARY KEY AUTOINCREMENT, author_name TEXT, pen_name TEXT, password TEXT, author_email TEXT, author_infos TEXT);`,
		`CREATE TABLE authors (author_id INTEGER PRIMARY KEY AUTOINCREMENT, author_name TEXT, pen_name TEXT, password TEXT, author_email TEXT, author_infos TEXT);`,
		`INSERT INTO global_config (id, is_init, site_name) VALUES (1, 0, 'Ubik');`,
	}

	for _, stmt := range schema {
		if execErr := db.Exec(stmt).Error; execErr != nil {
			t.Fatalf("create schema failed: %v", execErr)
		}
	}

	postgresDB = db
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestAdminAndSubmissionFunctions(t *testing.T) {
	db := setupTestDB(t)

	if err := db.Exec(`INSERT INTO admins (admin_name, password) VALUES ('superadmin', 'old')`).Error; err != nil {
		t.Fatalf("seed admin failed: %v", err)
	}

	admin, err := FindAdminByUsername("superadmin")
	if err != nil {
		t.Fatalf("FindAdminByUsername failed: %v", err)
	}
	if admin.AdminName != "superadmin" {
		t.Fatalf("unexpected admin: %+v", admin)
	}

	if err := ChangeAdminPassword(admin.AdminID, "new-pass"); err != nil {
		t.Fatalf("ChangeAdminPassword failed: %v", err)
	}

	contest := &model.Contest{
		ContestName:      "Contest A",
		ContestStartDate: datatypes.Date(time.Now()),
		ContestEndDate:   datatypes.Date(time.Now().Add(24 * time.Hour)),
	}
	if err := CreateContest(contest); err != nil {
		t.Fatalf("CreateContest failed: %v", err)
	}
	if contest.ContestID == 0 {
		t.Fatal("contest id should be assigned")
	}

	if err := UpdateContest(contest.ContestID, &model.Contest{ContestName: "Contest B"}); err != nil {
		t.Fatalf("UpdateContest failed: %v", err)
	}

	track := &model.Track{TrackName: "Track A", ContestID: contest.ContestID}
	if err := CreateTrack(track); err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}
	if track.TrackID == 0 {
		t.Fatal("track id should be assigned")
	}

	if err := UpdateTrack(track.TrackID, &model.Track{TrackName: "Track B"}); err != nil {
		t.Fatalf("UpdateTrack failed: %v", err)
	}

	if err := CreateActionLog(model.ActionLog{AdminID: admin.AdminID, Resource: "works", Action: "delete", Details: map[string]interface{}{"x": "y"}}); err != nil {
		t.Fatalf("CreateActionLog failed: %v", err)
	}

	work := &model.Work{WorkTitle: "Work A", TrackID: track.TrackID, AuthorID: 1}
	if err := SubmissionWork(work); err != nil {
		t.Fatalf("SubmissionWork failed: %v", err)
	}

	loaded := model.Work{WorkID: work.WorkID}
	if err := GetSubmissionByWorkID(&loaded); err != nil {
		t.Fatalf("GetSubmissionByWorkID failed: %v", err)
	}
	if loaded.WorkTitle != "Work A" {
		t.Fatalf("unexpected loaded work: %+v", loaded)
	}

	worksByTitle, err := FindWorksByWorkTitle("Work A")
	if err != nil || len(worksByTitle) != 1 {
		t.Fatalf("FindWorksByWorkTitle failed: %v len=%d", err, len(worksByTitle))
	}

	worksByTrack, err := GetWorksByTrackID(track.TrackID)
	if err != nil || len(worksByTrack) != 1 {
		t.Fatalf("GetWorksByTrackID failed: %v len=%d", err, len(worksByTrack))
	}

	worksByAuthor, err := GetWorksByAuthorID(1)
	if err != nil || len(worksByAuthor) != 1 {
		t.Fatalf("GetWorksByAuthorID failed: %v len=%d", err, len(worksByAuthor))
	}

	work.WorkTitle = "Work Updated"
	if err := UpdateWork(work); err != nil {
		t.Fatalf("UpdateWork failed: %v", err)
	}

	adminWork, err := GetWorkByID(work.WorkID)
	if err != nil {
		t.Fatalf("GetWorkByID failed: %v", err)
	}
	if adminWork.WorkTitle != "Work Updated" {
		t.Fatalf("GetWorkByID returned unexpected title: %+v", adminWork)
	}

	if err := DeleteWorkByID(work.WorkID); err != nil {
		t.Fatalf("DeleteWorkByID failed: %v", err)
	}
	if err := DeleteWorkByID(work.WorkID); err == nil {
		t.Fatal("DeleteWorkByID should fail for non-existing work")
	}

	if _, err := DeleteTrack(track.TrackID); err != nil {
		t.Fatalf("DeleteTrack failed: %v", err)
	}
	if _, err := DeleteContest(contest.ContestID); err != nil {
		t.Fatalf("DeleteContest failed: %v", err)
	}
}

func TestAuthorFunctions(t *testing.T) {
	db := setupTestDB(t)

	if err := db.Exec(`INSERT INTO author (author_name, password, author_email) VALUES ('alpha', 'p', 'a@example.com')`).Error; err != nil {
		t.Fatalf("seed author table failed: %v", err)
	}

	author := &model.Author{AuthorName: "alpha"}
	if err := GetAuthorByAuthorName(author); err != nil {
		t.Fatalf("GetAuthorByAuthorName failed: %v", err)
	}

	missing := &model.Author{AuthorName: "missing"}
	if err := GetAuthorByAuthorName(missing); err != _const.UsernameNotExist {
		t.Fatalf("expected UsernameNotExist, got %v", err)
	}

	var id int
	if err := db.Table("author").Select("author_id").Where("author_name = ?", "alpha").Scan(&id).Error; err != nil {
		t.Fatalf("query author id failed: %v", err)
	}

	authorByID := &model.Author{AuthorID: id}
	if err := GetAuthorByAuthorID(authorByID); err != nil {
		t.Fatalf("GetAuthorByAuthorID failed: %v", err)
	}

	created := &model.Author{AuthorName: "beta", Password: "p", AuthorEmail: "b@example.com"}
	if err := CreateAuthor(created); err != nil {
		t.Fatalf("CreateAuthor failed: %v", err)
	}
	created.PenName = "Beta"
	if err := UpdateAuthor(created); err != nil {
		t.Fatalf("UpdateAuthor failed: %v", err)
	}
	if err := DeleteAuthor(created); err != nil {
		t.Fatalf("DeleteAuthor failed: %v", err)
	}
}

func TestSystemFunctions(t *testing.T) {
	db := setupTestDB(t)

	cfg, err := getGlobalConfig()
	if err != nil {
		t.Fatalf("getGlobalConfig failed: %v", err)
	}
	if cfg.ID != 1 {
		t.Fatalf("unexpected global config id: %d", cfg.ID)
	}

	isInit, err := CheckIfSystemInit()
	if err != nil || isInit {
		t.Fatalf("CheckIfSystemInit unexpected result: %v, %v", isInit, err)
	}

	if err := ChangeSystemInitStatus(true); err != nil {
		t.Fatalf("ChangeSystemInitStatus failed: %v", err)
	}
	isInit, err = CheckIfSystemInit()
	if err != nil || !isInit {
		t.Fatalf("CheckIfSystemInit should become true: %v, %v", isInit, err)
	}

	emailConf := conf.EmailConfig{EmailAddress: "x@test.com", EmailAPPPassword: "pwd", SMTPHost: "smtp.test", SMTPPort: 587}
	if err := WriteSystemEmailConfig(emailConf); err != nil {
		t.Fatalf("WriteSystemEmailConfig failed: %v", err)
	}

	c1 := model.Contest{ContestName: "C1", ContestStartDate: datatypes.Date(time.Now()), ContestEndDate: datatypes.Date(time.Now())}
	c2 := model.Contest{ContestName: "C2", ContestStartDate: datatypes.Date(time.Now()), ContestEndDate: datatypes.Date(time.Now())}
	if err := db.Create(&c1).Error; err != nil {
		t.Fatalf("create contest c1 failed: %v", err)
	}
	if err := db.Create(&c2).Error; err != nil {
		t.Fatalf("create contest c2 failed: %v", err)
	}

	t1 := model.Track{TrackName: "T1", ContestID: c1.ContestID}
	t2 := model.Track{TrackName: "T2", ContestID: c1.ContestID}
	if err := db.Create(&t1).Error; err != nil {
		t.Fatalf("create track t1 failed: %v", err)
	}
	if err := db.Create(&t2).Error; err != nil {
		t.Fatalf("create track t2 failed: %v", err)
	}

	contests, err := GetContestList()
	if err != nil || len(contests) < 2 {
		t.Fatalf("GetContestList failed: %v len=%d", err, len(contests))
	}

	tracks, err := GetTrackList(c1.ContestID)
	if err != nil || len(tracks) != 2 {
		t.Fatalf("GetTrackList failed: %v len=%d", err, len(tracks))
	}

	if _, err := GetContestByID(c1.ContestID); err != nil {
		t.Fatalf("GetContestByID failed: %v", err)
	}
	if _, err := GetTrackByID(t1.TrackID); err != nil {
		t.Fatalf("GetTrackByID failed: %v", err)
	}

	tracksByContest, err := GetTracksByContestID(c1.ContestID)
	if err != nil || len(tracksByContest) != 2 {
		t.Fatalf("GetTracksByContestID failed: %v len=%d", err, len(tracksByContest))
	}

	if err := db.Exec(`DELETE FROM global_config`).Error; err != nil {
		t.Fatalf("delete global config failed: %v", err)
	}
	if _, err := getGlobalConfig(); err == nil {
		t.Fatal("getGlobalConfig should fail when row is missing")
	}
}

func TestScriptFlowFunctions(t *testing.T) {
	db := setupTestDB(t)

	if err := db.Exec(`INSERT INTO admins (admin_name, password) VALUES ('ops', 'pwd')`).Error; err != nil {
		t.Fatalf("seed admin failed: %v", err)
	}

	def := &model.ScriptDefinition{
		ScriptKey:   "submission_guard",
		ScriptName:  "Submission Guard",
		Interpreter: "python3",
		IsEnabled:   true,
		Meta:        map[string]any{"k": "v"},
	}
	if err := CreateScriptDefinition(def); err != nil {
		t.Fatalf("CreateScriptDefinition failed: %v", err)
	}
	if def.ScriptID == 0 {
		t.Fatal("script id should be assigned")
	}

	if _, err := GetScriptDefinitionByID(def.ScriptID); err != nil {
		t.Fatalf("GetScriptDefinitionByID failed: %v", err)
	}
	if _, err := GetScriptDefinitionByKey(def.ScriptKey); err != nil {
		t.Fatalf("GetScriptDefinitionByKey failed: %v", err)
	}

	if err := UpdateScriptDefinition(def.ScriptID, &model.ScriptDefinition{ScriptName: "Submission Guard V2", Interpreter: "python3"}); err != nil {
		t.Fatalf("UpdateScriptDefinition failed: %v", err)
	}
	if err := SetScriptDefinitionEnabled(def.ScriptID, true); err != nil {
		t.Fatalf("SetScriptDefinitionEnabled failed: %v", err)
	}

	nextVersion, err := GetNextScriptVersionNumber(def.ScriptID)
	if err != nil || nextVersion != 1 {
		t.Fatalf("GetNextScriptVersionNumber failed: %v next=%d", err, nextVersion)
	}

	v1 := &model.ScriptVersion{ScriptID: def.ScriptID, VersionNum: 1, FileName: "v1.py", RelativePath: "scripts/submission_guard/v1/v1.py", IsActive: true, CreatedBy: 1}
	if err = CreateScriptVersion(v1); err != nil {
		t.Fatalf("CreateScriptVersion v1 failed: %v", err)
	}
	v2 := &model.ScriptVersion{ScriptID: def.ScriptID, VersionNum: 2, FileName: "v2.py", RelativePath: "scripts/submission_guard/v2/v2.py", IsActive: false, CreatedBy: 1}
	if err = CreateScriptVersion(v2); err != nil {
		t.Fatalf("CreateScriptVersion v2 failed: %v", err)
	}

	if _, err = GetScriptVersionByID(v1.VersionID); err != nil {
		t.Fatalf("GetScriptVersionByID failed: %v", err)
	}
	versions, err := ListScriptVersions(def.ScriptID)
	if err != nil || len(versions) != 2 {
		t.Fatalf("ListScriptVersions failed: %v len=%d", err, len(versions))
	}

	if err = ActivateScriptVersion(def.ScriptID, v2.VersionID); err != nil {
		t.Fatalf("ActivateScriptVersion failed: %v", err)
	}
	active, err := GetActiveScriptVersion(def.ScriptID)
	if err != nil {
		t.Fatalf("GetActiveScriptVersion failed: %v", err)
	}
	if active.VersionID != v2.VersionID {
		t.Fatalf("unexpected active version: %+v", active)
	}

	flow := &model.ScriptFlow{FlowKey: "submission_pre_chain", FlowName: "Submission Pre Chain", IsEnabled: true, Meta: map[string]any{"x": 1}}
	if err = CreateScriptFlow(flow); err != nil {
		t.Fatalf("CreateScriptFlow failed: %v", err)
	}
	if flow.FlowID == 0 {
		t.Fatal("flow id should be assigned")
	}

	if err = UpdateScriptFlow(flow.FlowID, &model.ScriptFlow{FlowName: "Submission Pre Chain V2"}); err != nil {
		t.Fatalf("UpdateScriptFlow failed: %v", err)
	}
	if _, err = GetScriptFlowByID(flow.FlowID); err != nil {
		t.Fatalf("GetScriptFlowByID failed: %v", err)
	}
	flows, err := ListScriptFlows()
	if err != nil || len(flows) == 0 {
		t.Fatalf("ListScriptFlows failed: %v len=%d", err, len(flows))
	}
	if err = SetScriptFlowEnabled(flow.FlowID, true); err != nil {
		t.Fatalf("SetScriptFlowEnabled failed: %v", err)
	}

	steps := []model.FlowStep{{
		FlowID:          flow.FlowID,
		StepOrder:       1,
		StepName:        "guard",
		ScriptID:        def.ScriptID,
		ScriptVersionID: 0,
		TimeoutMs:       1000,
		FailureStrategy: "fail_close",
		InputTemplate:   map[string]any{"mode": "strict"},
		IsEnabled:       true,
	}}
	if err = ReplaceFlowSteps(flow.FlowID, steps); err != nil {
		t.Fatalf("ReplaceFlowSteps failed: %v", err)
	}
	listedSteps, err := ListFlowSteps(flow.FlowID)
	if err != nil || len(listedSteps) != 1 {
		t.Fatalf("ListFlowSteps failed: %v len=%d", err, len(listedSteps))
	}

	mount := &model.FlowMount{FlowID: flow.FlowID, Scope: "submission", EventKey: "file_pre", TargetType: "global", TargetID: 0, IsEnabled: true}
	if err = CreateFlowMount(mount); err != nil {
		t.Fatalf("CreateFlowMount failed: %v", err)
	}
	mounts, err := ListFlowMountsByFlow(flow.FlowID)
	if err != nil || len(mounts) != 1 {
		t.Fatalf("ListFlowMountsByFlow failed: %v len=%d", err, len(mounts))
	}

	resolvedFlow, resolvedSteps, err := ResolveFlowForExecution("submission", "file_pre", "track", 999)
	if err != nil {
		t.Fatalf("ResolveFlowForExecution failed: %v", err)
	}
	if resolvedFlow.FlowID != flow.FlowID || len(resolvedSteps) != 1 {
		t.Fatalf("unexpected resolved data: flow=%+v steps=%d", resolvedFlow, len(resolvedSteps))
	}
	if resolvedSteps[0].Version.VersionID != v2.VersionID {
		t.Fatalf("expected active version v2, got %+v", resolvedSteps[0].Version)
	}

	if err = DeleteFlowMount(mount.MountID); err != nil {
		t.Fatalf("DeleteFlowMount failed: %v", err)
	}
	if _, _, err = ResolveFlowForExecution("submission", "file_pre", "track", 999); err != ErrFlowNotMounted {
		t.Fatalf("expected ErrFlowNotMounted, got %v", err)
	}
}

func TestErrorBranches(t *testing.T) {
	t.Run("admin table errors", func(t *testing.T) {
		db := setupTestDB(t)
		if err := db.Exec(`DROP TABLE admins`).Error; err != nil {
			t.Fatalf("drop admins failed: %v", err)
		}

		if _, err := FindAdminByUsername("x"); err == nil {
			t.Fatal("FindAdminByUsername should fail when table is missing")
		}
		if err := ChangeAdminPassword(1, "x"); err == nil {
			t.Fatal("ChangeAdminPassword should fail when table is missing")
		}
	})

	t.Run("contest table errors", func(t *testing.T) {
		db := setupTestDB(t)
		if err := db.Exec(`DROP TABLE contests`).Error; err != nil {
			t.Fatalf("drop contests failed: %v", err)
		}

		contest := &model.Contest{ContestName: "x"}
		if err := CreateContest(contest); err == nil {
			t.Fatal("CreateContest should fail when table is missing")
		}
		if err := UpdateContest(1, &model.Contest{ContestName: "x"}); err == nil {
			t.Fatal("UpdateContest should fail when table is missing")
		}
		if _, err := DeleteContest(1); err == nil {
			t.Fatal("DeleteContest should fail when table is missing")
		}
		if _, err := GetContestByID(1); err == nil {
			t.Fatal("GetContestByID should fail when table is missing")
		}
		if _, err := GetContestList(); err == nil {
			t.Fatal("GetContestList should fail when table is missing")
		}
	})

	t.Run("track table errors", func(t *testing.T) {
		db := setupTestDB(t)
		if err := db.Exec(`DROP TABLE tracks`).Error; err != nil {
			t.Fatalf("drop tracks failed: %v", err)
		}

		track := &model.Track{TrackName: "x"}
		if err := CreateTrack(track); err == nil {
			t.Fatal("CreateTrack should fail when table is missing")
		}
		if err := UpdateTrack(1, &model.Track{TrackName: "x"}); err == nil {
			t.Fatal("UpdateTrack should fail when table is missing")
		}
		if _, err := DeleteTrack(1); err == nil {
			t.Fatal("DeleteTrack should fail when table is missing")
		}
		if _, err := GetTrackByID(1); err == nil {
			t.Fatal("GetTrackByID should fail when table is missing")
		}
		if _, err := GetTrackList(1); err == nil {
			t.Fatal("GetTrackList should fail when table is missing")
		}
		if _, err := GetTracksByContestID(1); err == nil {
			t.Fatal("GetTracksByContestID should fail when table is missing")
		}
	})

	t.Run("work table errors", func(t *testing.T) {
		db := setupTestDB(t)
		if err := db.Exec(`DROP TABLE works`).Error; err != nil {
			t.Fatalf("drop works failed: %v", err)
		}

		work := &model.Work{WorkTitle: "x"}
		if err := SubmissionWork(work); err == nil {
			t.Fatal("SubmissionWork should fail when table is missing")
		}
		if err := GetSubmissionByWorkID(&model.Work{WorkID: 1}); err == nil {
			t.Fatal("GetSubmissionByWorkID should fail when table is missing")
		}
		if _, err := FindWorksByWorkTitle("x"); err == nil {
			t.Fatal("FindWorksByWorkTitle should fail when table is missing")
		}
		if _, err := GetWorksByTrackID(1); err == nil {
			t.Fatal("GetWorksByTrackID should fail when table is missing")
		}
		if _, err := GetWorksByAuthorID(1); err == nil {
			t.Fatal("GetWorksByAuthorID should fail when table is missing")
		}
		if err := UpdateWork(&model.Work{WorkID: 1}); err == nil {
			t.Fatal("UpdateWork should fail when table is missing")
		}
		if err := DeleteWork(&model.Work{WorkID: 1}); err == nil {
			t.Fatal("DeleteWork should fail when table is missing")
		}
		if _, err := GetWorkByID(1); err == nil {
			t.Fatal("GetWorkByID should fail when table is missing")
		}
		if err := DeleteWorkByID(1); err == nil {
			t.Fatal("DeleteWorkByID should fail when table is missing")
		}
	})

	t.Run("author table errors", func(t *testing.T) {
		db := setupTestDB(t)
		if err := db.Exec(`DROP TABLE author`).Error; err != nil {
			t.Fatalf("drop author failed: %v", err)
		}
		if err := GetAuthorByAuthorName(&model.Author{AuthorName: "x"}); err == nil {
			t.Fatal("GetAuthorByAuthorName should fail when table is missing")
		}
		if err := GetAuthorByAuthorID(&model.Author{AuthorID: 1}); err == nil {
			t.Fatal("GetAuthorByAuthorID should fail when table is missing")
		}

		if err := db.Exec(`DROP TABLE authors`).Error; err != nil {
			t.Fatalf("drop authors failed: %v", err)
		}
		if err := CreateAuthor(&model.Author{AuthorName: "x"}); err == nil {
			t.Fatal("CreateAuthor should fail when table is missing")
		}
		if err := UpdateAuthor(&model.Author{AuthorID: 1, AuthorName: "n"}); err == nil {
			t.Fatal("UpdateAuthor should fail when table is missing")
		}
		if err := DeleteAuthor(&model.Author{AuthorID: 1}); err == nil {
			t.Fatal("DeleteAuthor should fail when table is missing")
		}
	})

	t.Run("action log and config errors", func(t *testing.T) {
		db := setupTestDB(t)
		if err := db.Exec(`DROP TABLE action_logs`).Error; err != nil {
			t.Fatalf("drop action_logs failed: %v", err)
		}
		if err := CreateActionLog(model.ActionLog{AdminID: 1, Resource: "x", Action: "x"}); err == nil {
			t.Fatal("CreateActionLog should fail when table is missing")
		}

		if err := db.Exec(`DROP TABLE global_config`).Error; err != nil {
			t.Fatalf("drop global_config failed: %v", err)
		}
		if _, err := CheckIfSystemInit(); err == nil {
			t.Fatal("CheckIfSystemInit should fail when table is missing")
		}
		if err := ChangeSystemInitStatus(true); err == nil {
			t.Fatal("ChangeSystemInitStatus should fail when table is missing")
		}
		if err := WriteSystemEmailConfig(conf.EmailConfig{EmailAddress: "a", SMTPHost: "b"}); err == nil {
			t.Fatal("WriteSystemEmailConfig should fail when table is missing")
		}
	})
}

func TestStartInvocation(t *testing.T) {
	_ = Start(conf.DBConfig{Host: "127.0.0.1", Port: "abc", User: "u", Password: "p", LogLevel: 1})
}
