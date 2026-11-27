package launcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"t-deck/internal/fsutil"
)

// SuggestChangelog reads git commit history since the latest tag (or last 15 commits) to prefill release notes.
func SuggestChangelog(localDir string) (string, error) {
	if strings.TrimSpace(localDir) == "" {
		return "", errors.New("chưa có thư mục mã nguồn cục bộ")
	}
	if _, err := os.Stat(filepath.Join(localDir, ".git")); err != nil {
		return "", errors.New("thư mục dự án không phải Git repository")
	}

	rangeSpec := ""
	tagCmd := exec.Command("git", "-C", localDir, "describe", "--tags", "--abbrev=0")
	hideConsoleWindow(tagCmd)
	if tagOut, err := tagCmd.Output(); err == nil {
		tag := strings.TrimSpace(string(tagOut))
		if tag != "" {
			rangeSpec = tag + "..HEAD"
		}
	}

	args := []string{"-C", localDir, "log"}
	if rangeSpec != "" {
		args = append(args, rangeSpec)
	}
	args = append(args, "--pretty=format:- %s (%h)", "-n", "25")
	logCmd := exec.Command("git", args...)
	hideConsoleWindow(logCmd)
	out, err := logCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git log failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	notes := strings.TrimSpace(string(out))
	if notes == "" {
		return "- Cập nhật phiên bản mới", nil
	}
	return notes, nil
}

// RunBuildCommand executes cmdStr inside workDir with context cancellation and streamed output lines.
func RunBuildCommand(ctx context.Context, workDir string, cmdStr string, onLine func(stream, line string)) error {
	cleanCmd := strings.TrimSpace(cmdStr)
	if cleanCmd == "" {
		return errors.New("chưa cấu hình build_cmd cho target này")
	}
	if strings.TrimSpace(workDir) == "" {
		return errors.New("thư mục build không hợp lệ")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/C", cleanCmd)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", cleanCmd)
	}
	cmd.Dir = workDir
	hideConsoleWindow(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("không thể khởi chạy build command: %w", err)
	}

	var wg sync.WaitGroup
	streamPipe := func(streamName string, r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 512*1024)
		for sc.Scan() {
			if onLine != nil {
				onLine(streamName, sc.Text())
			}
		}
	}
	wg.Add(2)
	go streamPipe("stdout", stdout)
	go streamPipe("stderr", stderr)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("build thất bại: %w", err)
	}
	return nil
}

// ResolveArtifactGlob finds the newest file or directory matching pattern relative to baseDir.
func ResolveArtifactGlob(baseDir, pattern string) (string, error) {
	pat := strings.TrimSpace(pattern)
	if pat == "" {
		return "", errors.New("artifact pattern trống")
	}
	isRel := !filepath.IsAbs(pat)
	fullPat := pat
	if isRel {
		fullPat = filepath.Join(baseDir, pat)
	}
	matches, err := filepath.Glob(fullPat)
	if err != nil {
		return "", fmt.Errorf("invalid glob %q: %w", pat, err)
	}
	if isRel && len(matches) > 0 {
		safeMatches := make([]string, 0, len(matches))
		for _, m := range matches {
			if _, safeErr := fsutil.ResolveSafeSubpath(baseDir, m); safeErr == nil {
				safeMatches = append(safeMatches, m)
			}
		}
		matches = safeMatches
	}
	if len(matches) == 0 {
		if isRel {
			if _, safeErr := fsutil.ResolveSafeSubpath(baseDir, fullPat); safeErr != nil {
				return "", safeErr
			}
		}
		// Direct stat fallback if pattern has no wildcards or path exists literally.
		if _, statErr := os.Stat(fullPat); statErr == nil {
			return fullPat, nil
		}
		return "", fmt.Errorf("không tìm thấy artifact khớp %q", fullPat)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	type candidate struct {
		path    string
		modTime time.Time
	}
	cands := make([]candidate, 0, len(matches))
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil {
			cands = append(cands, candidate{path: m, modTime: info.ModTime()})
		}
	}
	if len(cands) == 0 {
		return matches[0], nil
	}
	sort.Slice(cands, func(i, j int) bool {
		return cands[i].modTime.After(cands[j].modTime)
	})
	return cands[0].path, nil
}
