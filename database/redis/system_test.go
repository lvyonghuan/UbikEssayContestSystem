package redis

import (
	"main/conf"
	"main/model"
	_const "main/util/const"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func initTestRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}

	host, portStr, err := net.SplitHostPort(mr.Addr())
	if err != nil {
		mr.Close()
		t.Fatalf("split host port failed: %v", err)
	}
	if _, err := strconv.Atoi(portStr); err != nil {
		mr.Close()
		t.Fatalf("invalid redis port: %v", err)
	}

	if err := InitRedis(conf.RedisConfig{Host: host, Port: portStr, Password: "", DB: 0}); err != nil {
		mr.Close()
		t.Fatalf("InitRedis failed: %v", err)
	}

	t.Cleanup(func() {
		mr.Close()
	})

	return mr
}

func TestTrackCacheLifecycle(t *testing.T) {
	initTestRedis(t)

	contest := model.Contest{
		ContestStartDate: time.Unix(1000, 0),
		ContestEndDate:   time.Unix(2000, 0),
	}
	track := model.Track{TrackID: 9}

	if err := CreateTrack(track, contest); err != nil {
		t.Fatalf("CreateTrack failed: %v", err)
	}

	start, end, err := GetStartAndEndDate(track.TrackID)
	if err != nil {
		t.Fatalf("GetStartAndEndDate failed: %v", err)
	}

	expStart := contest.ContestStartDate.Unix()
	expEnd := contest.ContestEndDate.Unix()
	if start != expStart || end != expEnd {
		t.Fatalf("unexpected start/end: got (%d,%d) expect (%d,%d)", start, end, expStart, expEnd)
	}

	if err := DeleteTrack(track.TrackID); err != nil {
		t.Fatalf("DeleteTrack failed: %v", err)
	}
	if _, _, err := GetStartAndEndDate(track.TrackID); err == nil {
		t.Fatal("GetStartAndEndDate should fail after DeleteTrack")
	}
}

func TestUploadPermissionLifecycle(t *testing.T) {
	initTestRedis(t)

	if err := SetUploadFilePermission(11, 22, 33); err != nil {
		t.Fatalf("SetUploadFilePermission failed: %v", err)
	}

	authorID, trackID, err := GetUploadFilePermission(33)
	if err != nil {
		t.Fatalf("GetUploadFilePermission failed: %v", err)
	}
	if authorID != 11 || trackID != 22 {
		t.Fatalf("unexpected permission value: author=%d track=%d", authorID, trackID)
	}

	if err := DeleteUploadFilePermission(33); err != nil {
		t.Fatalf("DeleteUploadFilePermission failed: %v", err)
	}
	if _, _, err := GetUploadFilePermission(33); err == nil {
		t.Fatal("GetUploadFilePermission should fail after delete")
	}
}

func TestUploadPermissionInvalidFormat(t *testing.T) {
	initTestRedis(t)

	key := _const.RedisUploadPermissionPrefix + "-100"
	if err := rdb.client.Set(rdb.ctx, key, "invalid-value", 0).Err(); err != nil {
		t.Fatalf("seed invalid permission failed: %v", err)
	}

	if _, _, err := GetUploadFilePermission(100); err == nil {
		t.Fatal("GetUploadFilePermission should fail on invalid value format")
	}
}

func TestGetStartAndEndDateInvalidNumber(t *testing.T) {
	initTestRedis(t)

	startKey := _const.RedisTrackStartDatePrefix + "-1"
	endKey := _const.RedisTrackEndDatePrefix + "-1"
	if err := rdb.client.Set(rdb.ctx, startKey, "bad-int", 0).Err(); err != nil {
		t.Fatalf("seed bad start failed: %v", err)
	}
	if err := rdb.client.Set(rdb.ctx, endKey, "123", 0).Err(); err != nil {
		t.Fatalf("seed end failed: %v", err)
	}

	if _, _, err := GetStartAndEndDate(-1); err == nil {
		t.Fatal("GetStartAndEndDate should fail when start value is not integer")
	}
}

func TestGetUploadPermissionInvalidNumberParts(t *testing.T) {
	initTestRedis(t)

	key := _const.RedisUploadPermissionPrefix + "-200"
	if err := rdb.client.Set(rdb.ctx, key, "a-2", 0).Err(); err != nil {
		t.Fatalf("seed invalid author id failed: %v", err)
	}
	if _, _, err := GetUploadFilePermission(200); err == nil {
		t.Fatal("GetUploadFilePermission should fail when author id is invalid")
	}

	if err := rdb.client.Set(rdb.ctx, key, "1-b", 0).Err(); err != nil {
		t.Fatalf("seed invalid track id failed: %v", err)
	}
	if _, _, err := GetUploadFilePermission(200); err == nil {
		t.Fatal("GetUploadFilePermission should fail when track id is invalid")
	}
}

func TestInitRedisFailure(t *testing.T) {
	if err := InitRedis(conf.RedisConfig{Host: "127.0.0.1", Port: "1", Password: "", DB: 0}); err == nil {
		t.Fatal("InitRedis should fail for unreachable redis")
	}
}
