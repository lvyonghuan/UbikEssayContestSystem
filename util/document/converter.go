package document

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Converter interface {
	ConvertDocToDocx(ctx context.Context, srcDocPath string, dstDocxPath string) error
}

type LibreOfficeConverter struct {
	officeBinary string
	tempRoot     string
	timeout      time.Duration
}

func NewLibreOfficeConverter(officeBinary string, tempRoot string, timeout time.Duration) *LibreOfficeConverter {
	if strings.TrimSpace(officeBinary) == "" {
		officeBinary = "soffice"
	}
	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	return &LibreOfficeConverter{
		officeBinary: officeBinary,
		tempRoot:     tempRoot,
		timeout:      timeout,
	}
}

func (c *LibreOfficeConverter) ConvertDocToDocx(ctx context.Context, srcDocPath string, dstDocxPath string) error {
	sourceExt := strings.ToLower(filepath.Ext(srcDocPath))
	if sourceExt != ".doc" {
		return errors.New("source file must be .doc")
	}

	if strings.ToLower(filepath.Ext(dstDocxPath)) != ".docx" {
		return errors.New("destination file must be .docx")
	}

	err := os.MkdirAll(filepath.Dir(dstDocxPath), os.ModePerm)
	if err != nil {
		return err
	}

	workDir, err := os.MkdirTemp(c.tempRoot, "doc-convert-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	stepCtx := ctx
	if stepCtx == nil {
		stepCtx = context.Background()
	}
	stepCtx, cancel := context.WithTimeout(stepCtx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(
		stepCtx,
		c.officeBinary,
		"--headless",
		"--convert-to",
		"docx",
		"--outdir",
		workDir,
		srcDocPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("libreoffice convert failed: %w, output: %s", err, strings.TrimSpace(string(output)))
	}

	generated := filepath.Join(workDir, strings.TrimSuffix(filepath.Base(srcDocPath), filepath.Ext(srcDocPath))+".docx")
	if _, err = os.Stat(generated); err != nil {
		return fmt.Errorf("docx not generated: %w", err)
	}

	return moveFile(generated, dstDocxPath)
}

func moveFile(srcPath string, dstPath string) error {
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

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Do not remove srcPath here. The caller owns the temp working directory
	// lifecycle and may clean it later; immediate removal can fail on Windows
	// due to short-lived file locks from external converters.
	return nil
}
