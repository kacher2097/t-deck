package cloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"t-deck/internal/model"
)

var (
	sharedHealthClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	reMetaAppVersion = regexp.MustCompile(`(?i)<meta[^>]+name=["']app-version["'][^>]+content=["']([^"']+)["']`)
)

// CheckWebHealth checks HTTP liveness, latency, and optional version drift for a Web target.
func CheckWebHealth(ctx context.Context, target model.ProjectTarget, latestReleaseVersion string) model.WebHealthResult {
	res := model.WebHealthResult{
		TargetID:  target.ID,
		ProjectID: target.ProjectID,
		CheckedAt: time.Now().Unix(),
	}

	probeURL := strings.TrimSpace(target.HealthURL)
	if probeURL == "" {
		probeURL = strings.TrimSpace(target.ProdURL)
	}
	if probeURL == "" {
		res.Error = "chưa cấu hình URL kiểm tra"
		return res
	}
	if !isSafeHTTPURL(probeURL) {
		res.Error = "URL không hợp lệ (chỉ hỗ trợ http/https)"
		return res
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	resp, err := sharedHealthClient.Do(req)
	res.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
	_ = resp.Body.Close()

	res.StatusCode = resp.StatusCode
	res.Online = resp.StatusCode >= 200 && resp.StatusCode < 400

	versionURL := strings.TrimSpace(target.VersionURL)
	if versionURL != "" && isSafeHTTPURL(versionURL) {
		if ver, vErr := fetchLiveVersionJSON(ctx, versionURL); vErr == nil && ver != "" {
			res.LiveVersion = ver
		}
	} else if len(bodyBytes) > 0 {
		if m := reMetaAppVersion.FindSubmatch(bodyBytes); len(m) == 2 {
			res.LiveVersion = strings.TrimSpace(string(m[1]))
		} else {
			res.LiveVersion = extractVersionFromJSON(bodyBytes)
		}
	}

	if res.LiveVersion != "" && strings.TrimSpace(latestReleaseVersion) != "" {
		res.Drift = normalizeVersion(res.LiveVersion) != normalizeVersion(latestReleaseVersion)
	}
	return res
}

// CheckAllWebHealth concurrently probes a slice of Web targets using a bounded worker pool.
func CheckAllWebHealth(
	ctx context.Context,
	targets []model.ProjectTarget,
	latestByTarget map[string]string,
	onResult func(model.WebHealthResult),
) []model.WebHealthResult {
	var webTargets []model.ProjectTarget
	for _, t := range targets {
		if t.Kind == "web" && (strings.TrimSpace(t.HealthURL) != "" || strings.TrimSpace(t.ProdURL) != "") {
			webTargets = append(webTargets, t)
		}
	}
	if len(webTargets) == 0 {
		return []model.WebHealthResult{}
	}

	results := make([]model.WebHealthResult, len(webTargets))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup

	for i, tgt := range webTargets {
		i, tgt := i, tgt
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r := CheckWebHealth(ctx, tgt, latestByTarget[tgt.ID])
			results[i] = r
			if onResult != nil {
				onResult(r)
			}
		}()
	}
	wg.Wait()
	return results
}

func fetchLiveVersionJSON(ctx context.Context, versionURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := sharedHealthClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}
	return extractVersionFromJSON(data), nil
}

func extractVersionFromJSON(data []byte) string {
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return ""
	}
	for _, k := range []string{"version", "app_version", "appVersion", "tag", "build"} {
		if raw, ok := parsed[k]; ok {
			if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func normalizeVersion(v string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(v), "v"))
}

func isSafeHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
