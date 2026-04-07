package log

import (
	"main/conf"
	"path/filepath"
	"testing"
)

func TestInitLoggr(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	InitLoggr(conf.LogConfig{Level: 1, WriteLevel: 1, LogFilePath: logPath})
	if Logger == nil {
		t.Fatal("Logger should be initialized")
	}
}
