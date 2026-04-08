package document

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConvertDocToDocxValidation(t *testing.T) {
	converter := NewLibreOfficeConverter("missing-binary", "", time.Second)

	err := converter.ConvertDocToDocx(context.Background(), "a.txt", "b.docx")
	if err == nil {
		t.Fatal("expected error for non-doc source")
	}

	err = converter.ConvertDocToDocx(context.Background(), "a.doc", "b.txt")
	if err == nil {
		t.Fatal("expected error for non-docx destination")
	}
}

func TestConvertDocToDocxBinaryError(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "input.doc")
	dst := filepath.Join(tmp, "output.docx")
	if err := os.WriteFile(src, []byte("doc"), 0o644); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	converter := NewLibreOfficeConverter("missing-binary", tmp, time.Second)
	if err := converter.ConvertDocToDocx(context.Background(), src, dst); err == nil {
		t.Fatal("expected convert failure for missing binary")
	}
}

func TestMoveFile(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "a.txt")
	dst := filepath.Join(tmp, "b.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src failed: %v", err)
	}

	if err := moveFile(src, dst); err != nil {
		t.Fatalf("moveFile failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst failed: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected dst content: %s", string(data))
	}
}
