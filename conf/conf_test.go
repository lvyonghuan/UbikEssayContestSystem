package conf

import (
	"log"
	"os"
	"testing"
)

func changeWorkDir() {
	if err := os.Chdir("../"); err != nil {
		panic(err)
	}
}

func TestReadConfig_Success(t *testing.T) {
	// 切换到项目根目录
	changeWorkDir()

	// 确保 conf 目录存在
	if err := os.MkdirAll("conf", 0o755); err != nil {
		t.Fatalf("failed to create conf dir: %v", err)
	}
	
	cfg, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig returned error: %v", err)
	}

	log.Println(cfg)
}
