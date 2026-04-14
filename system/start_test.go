package system

import (
	"errors"
	"main/conf"
	"main/model"
	"testing"
	"time"
)

func backupSystemStartHooks(t *testing.T) {
	origInitContestRedisCacheFn := initContestRedisCacheFn
	origInitContestEndBuiltinFlowFn := initContestEndBuiltinFlowFn
	origInitContestEndSchedulesFn := initContestEndSchedulesFn
	origInitGlobalInfoRouterFn := initGlobalInfoRouterFn
	origGetContestListForCacheFn := getContestListForCacheFn
	origGetTracksByContestFn := getTracksByContestFn
	origCreateTrackCacheFn := createTrackCacheFn

	t.Cleanup(func() {
		initContestRedisCacheFn = origInitContestRedisCacheFn
		initContestEndBuiltinFlowFn = origInitContestEndBuiltinFlowFn
		initContestEndSchedulesFn = origInitContestEndSchedulesFn
		initGlobalInfoRouterFn = origInitGlobalInfoRouterFn
		getContestListForCacheFn = origGetContestListForCacheFn
		getTracksByContestFn = origGetTracksByContestFn
		createTrackCacheFn = origCreateTrackCacheFn
	})
}

func TestInitContestRedisCacheSuccess(t *testing.T) {
	backupSystemStartHooks(t)

	created := 0
	getContestListForCacheFn = func() ([]model.Contest, error) {
		return []model.Contest{{ContestID: 1}, {ContestID: 2}}, nil
	}
	getTracksByContestFn = func(contestID int) ([]model.Track, error) {
		switch contestID {
		case 1:
			return []model.Track{{TrackID: 11, ContestID: 1}, {TrackID: 12, ContestID: 1}}, nil
		case 2:
			return []model.Track{{TrackID: 21, ContestID: 2}}, nil
		default:
			return nil, nil
		}
	}
	createTrackCacheFn = func(track model.Track, contest model.Contest) error {
		if track.ContestID != contest.ContestID {
			t.Fatalf("track/contest mismatch: %+v %+v", track, contest)
		}
		created++
		return nil
	}

	if err := initContestRedisCache(); err != nil {
		t.Fatalf("initContestRedisCache failed: %v", err)
	}
	if created != 3 {
		t.Fatalf("expected 3 track caches created, got %d", created)
	}
}

func TestInitContestRedisCacheErrors(t *testing.T) {
	backupSystemStartHooks(t)

	getContestListForCacheFn = func() ([]model.Contest, error) {
		return nil, errors.New("contest list error")
	}
	if err := initContestRedisCache(); err == nil {
		t.Fatal("initContestRedisCache should fail when contest list query fails")
	}

	getContestListForCacheFn = func() ([]model.Contest, error) {
		return []model.Contest{{ContestID: 1}}, nil
	}
	getTracksByContestFn = func(contestID int) ([]model.Track, error) {
		return nil, errors.New("track list error")
	}
	if err := initContestRedisCache(); err == nil {
		t.Fatal("initContestRedisCache should fail when track list query fails")
	}

	getTracksByContestFn = func(contestID int) ([]model.Track, error) {
		return []model.Track{{TrackID: 1, ContestID: contestID}}, nil
	}
	createTrackCacheFn = func(track model.Track, contest model.Contest) error {
		return errors.New("redis error")
	}
	if err := initContestRedisCache(); err == nil {
		t.Fatal("initContestRedisCache should fail when redis cache write fails")
	}
}

func TestSysStart(t *testing.T) {
	backupSystemStartHooks(t)

	cacheCalled := false
	builtinFlowCalled := false
	scheduleCalled := false
	routerCalled := make(chan struct{}, 1)
	initContestRedisCacheFn = func() error {
		cacheCalled = true
		return nil
	}
	initContestEndBuiltinFlowFn = func() error {
		builtinFlowCalled = true
		return nil
	}
	initContestEndSchedulesFn = func() error {
		scheduleCalled = true
		return nil
	}
	initGlobalInfoRouterFn = func(apiConf conf.APIConfig) {
		routerCalled <- struct{}{}
	}

	SysStart(conf.APIConfig{GlobalInfoPort: "18888"})
	if !cacheCalled {
		t.Fatal("SysStart should initialize contest redis cache")
	}
	if !builtinFlowCalled {
		t.Fatal("SysStart should initialize contest end builtin flow")
	}
	if !scheduleCalled {
		t.Fatal("SysStart should initialize contest end schedules")
	}

	select {
	case <-routerCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("SysStart should start global info router goroutine")
	}
}

func TestSysStartPanicOnCacheInitFailure(t *testing.T) {
	backupSystemStartHooks(t)

	initContestRedisCacheFn = func() error {
		return errors.New("boom")
	}
	initContestEndBuiltinFlowFn = func() error { return nil }

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("SysStart should panic when contest cache initialization fails")
		}
	}()

	SysStart(conf.APIConfig{})
}

func TestSysStartPanicOnScheduleInitFailure(t *testing.T) {
	backupSystemStartHooks(t)

	initContestRedisCacheFn = func() error { return nil }
	initContestEndBuiltinFlowFn = func() error { return nil }
	initContestEndSchedulesFn = func() error {
		return errors.New("boom schedule")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("SysStart should panic when contest end schedule initialization fails")
		}
	}()

	SysStart(conf.APIConfig{})
}

func TestSysStartPanicOnBuiltinFlowInitFailure(t *testing.T) {
	backupSystemStartHooks(t)

	initContestRedisCacheFn = func() error { return nil }
	initContestEndBuiltinFlowFn = func() error {
		return errors.New("boom builtin flow")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("SysStart should panic when contest end builtin flow initialization fails")
		}
	}()

	SysStart(conf.APIConfig{})
}
