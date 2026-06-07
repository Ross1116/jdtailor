package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var tectonicLatestReleaseURL = "https://api.github.com/repos/tectonic-typesetting/tectonic/releases/latest"

var execCommandContext = exec.CommandContext

type TectonicStatus struct {
	Status         string `json:"status"`
	ExecutablePath string `json:"executable_path"`
	Error          string `json:"error"`
}

type InstallTectonicResult struct {
	Success        bool   `json:"success"`
	Status         string `json:"status"`
	ExecutablePath string `json:"executable_path"`
	Error          string `json:"error"`
}

type RenderPDFResult struct {
	Success bool   `json:"success"`
	TexPath string `json:"tex_path"`
	PDFPath string `json:"pdf_path"`
	Error   string `json:"error"`
}

func (s *Store) TectonicStatus() TectonicStatus {
	path := s.tectonicPath()
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return TectonicStatus{Status: "missing", ExecutablePath: path}
	}
	if err != nil {
		return TectonicStatus{Status: "error", ExecutablePath: path, Error: err.Error()}
	}
	if info.IsDir() {
		return TectonicStatus{Status: "error", ExecutablePath: path, Error: "path is a directory"}
	}
	return TectonicStatus{Status: "installed", ExecutablePath: path}
}

func (s *Store) InstallTectonic(ctx context.Context) (InstallTectonicResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if runtime.GOOS != "windows" {
		result := InstallTectonicResult{
			Success:        false,
			Status:         "error",
			ExecutablePath: s.tectonicPath(),
			Error:          "automatic Tectonic install is only wired for Windows in this slice",
		}
		_ = s.LogEvent("error", "Tectonic install failed: "+result.Error)
		return result, nil
	}
	toolDir := filepath.Dir(s.tectonicPath())
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return InstallTectonicResult{}, err
	}
	zipPath := filepath.Join(toolDir, "tectonic.zip")
	client := &http.Client{Timeout: 2 * time.Minute}
	downloadURL, err := tectonicDownloadURL(ctx, client)
	if err != nil {
		result := InstallTectonicResult{Status: "error", ExecutablePath: s.tectonicPath(), Error: err.Error()}
		_ = s.LogEvent("error", "Tectonic install failed: "+err.Error())
		return result, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return InstallTectonicResult{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		result := InstallTectonicResult{Status: "error", ExecutablePath: s.tectonicPath(), Error: err.Error()}
		_ = s.LogEvent("error", "Tectonic install failed: "+err.Error())
		return result, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result := InstallTectonicResult{
			Status:         "error",
			ExecutablePath: s.tectonicPath(),
			Error:          fmt.Sprintf("download returned HTTP %d", resp.StatusCode),
		}
		_ = s.LogEvent("error", "Tectonic install failed: "+result.Error)
		return result, nil
	}
	out, err := os.Create(zipPath)
	if err != nil {
		return InstallTectonicResult{}, err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return InstallTectonicResult{}, err
	}
	if err := out.Close(); err != nil {
		return InstallTectonicResult{}, err
	}
	if err := unzipTectonic(zipPath, toolDir, s.tectonicPath()); err != nil {
		result := InstallTectonicResult{Status: "error", ExecutablePath: s.tectonicPath(), Error: err.Error()}
		_ = s.LogEvent("error", "Tectonic install failed: "+err.Error())
		return result, nil
	}
	_ = os.Remove(zipPath)
	result := InstallTectonicResult{Success: true, Status: "installed", ExecutablePath: s.tectonicPath()}
	_ = s.LogEvent("info", "Tectonic installed")
	return result, nil
}

func (s *Store) RenderSamplePDF(ctx context.Context) (RenderPDFResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	status := s.TectonicStatus()
	result := RenderPDFResult{}
	if status.Status != "installed" {
		result.Error = "Tectonic is " + status.Status
		_ = s.LogEvent("error", "sample PDF render failed: "+result.Error)
		return result, nil
	}
	outputDir := filepath.Join(s.generatedPath, "sample-pdf")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return RenderPDFResult{}, err
	}
	texPath := filepath.Join(outputDir, "sample.tex")
	pdfPath := filepath.Join(outputDir, "sample.pdf")
	if err := os.WriteFile(texPath, []byte(sampleTeX()), 0o644); err != nil {
		return RenderPDFResult{}, err
	}
	cmd := execCommandContext(ctx, status.ExecutablePath, "-X", "compile", "--outdir", outputDir, texPath)
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	result.TexPath = texPath
	result.PDFPath = pdfPath
	if err != nil {
		result.Error = strings.TrimSpace(string(output))
		if result.Error == "" {
			result.Error = err.Error()
		}
		_ = s.LogEvent("error", "sample PDF render failed: "+result.Error)
		return result, nil
	}
	if _, err := os.Stat(pdfPath); err != nil {
		result.Error = "PDF was not created"
		_ = s.LogEvent("error", "sample PDF render failed: "+result.Error)
		return result, nil
	}
	result.Success = true
	_ = s.LogEvent("info", "sample PDF rendered")
	return result, nil
}

func sampleTeX() string {
	return `\documentclass{article}
\begin{document}
JD Tailor PDF check
\end{document}
`
}

func unzipTectonic(zipPath string, destDir string, finalExe string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if !strings.EqualFold(filepath.Base(file.Name), "tectonic.exe") {
			continue
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()
		out, err := os.OpenFile(finalExe, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, src); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	}
	return errors.New("tectonic.exe not found in archive")
}

func tectonicDownloadURL(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tectonicLatestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("release lookup returned HTTP %d", resp.StatusCode)
	}
	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return "", err
	}
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if strings.HasSuffix(name, "x86_64-pc-windows-msvc.zip") && asset.BrowserDownloadURL != "" {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", errors.New("Windows Tectonic ZIP asset not found in latest release")
}

func (s *Store) tectonicPath() string {
	return filepath.Join(s.root, "tools", "tectonic", "tectonic.exe")
}
