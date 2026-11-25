package launcher

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"t-deck/internal/apperr"
	"t-deck/internal/model"
)

var reListenAddrPort = regexp.MustCompile(`(?:127\.0\.0\.1|0\.0\.0\.0|\[::1?\]|::):(\d+)$`)

// ProcessMonitor tracks launched child PIDs and detects their bound localhost listening TCP ports.
type ProcessMonitor struct {
	mu      sync.Mutex
	tracked map[string]*trackedProc
	onDelta func(model.RunningProcess)
	stopCh  chan struct{}
	once    sync.Once
}

type trackedProc struct {
	projectID string
	pid       int
	ports     []int
	startedAt int64
	command   string
}

// NewProcessMonitor creates and starts a ProcessMonitor that polls only while active PIDs are tracked.
func NewProcessMonitor(onDelta func(model.RunningProcess)) *ProcessMonitor {
	pm := &ProcessMonitor{
		tracked: make(map[string]*trackedProc),
		onDelta: onDelta,
		stopCh:  make(chan struct{}),
	}
	go pm.loop()
	return pm
}

// Track registers a newly launched project PID.
func (pm *ProcessMonitor) Track(projectID string, pid int, command string) {
	if pm == nil || pid <= 0 || strings.TrimSpace(projectID) == "" {
		return
	}
	pm.mu.Lock()
	entry := &trackedProc{
		projectID: projectID,
		pid:       pid,
		ports:     []int{},
		startedAt: time.Now().Unix(),
		command:   command,
	}
	pm.tracked[projectID] = entry
	pm.mu.Unlock()

	if pm.onDelta != nil {
		pm.onDelta(model.RunningProcess{
			ProjectID: entry.projectID,
			PID:       entry.pid,
			Ports:     []int{},
			StartedAt: entry.startedAt,
			Command:   entry.command,
		})
	}
}

// ListRunning returns all currently tracked running processes.
func (pm *ProcessMonitor) ListRunning() []model.RunningProcess {
	if pm == nil {
		return []model.RunningProcess{}
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	out := make([]model.RunningProcess, 0, len(pm.tracked))
	for _, t := range pm.tracked {
		portsCopy := append([]int(nil), t.ports...)
		out = append(out, model.RunningProcess{
			ProjectID: t.projectID,
			PID:       t.pid,
			Ports:     portsCopy,
			StartedAt: t.startedAt,
			Command:   t.command,
		})
	}
	return out
}

// StopProject terminates the tracked process tree for projectID via taskkill.
func (pm *ProcessMonitor) StopProject(projectID string) error {
	if pm == nil {
		return nil
	}
	pm.mu.Lock()
	t, ok := pm.tracked[projectID]
	if ok {
		delete(pm.tracked, projectID)
	}
	pm.mu.Unlock()

	if !ok || t.pid <= 0 {
		return nil
	}
	killCmd := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(t.pid))
	hideConsoleWindow(killCmd)
	_ = killCmd.Run()
	if pm.onDelta != nil {
		pm.onDelta(model.RunningProcess{
			ProjectID: projectID,
			PID:       0,
			Ports:     []int{},
		})
	}
	return nil
}

// Close stops the background polling goroutine cleanly.
func (pm *ProcessMonitor) Close() {
	if pm == nil {
		return
	}
	pm.once.Do(func() {
		close(pm.stopCh)
	})
}

func (pm *ProcessMonitor) loop() {
	defer apperr.Recover("process-monitor", nil, nil)
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-pm.stopCh:
			return
		case <-ticker.C:
			pm.safePollOnce()
		}
	}
}

func (pm *ProcessMonitor) safePollOnce() {
	defer apperr.Recover("process-monitor-poll", nil, nil)
	pm.pollOnce()
}

func (pm *ProcessMonitor) pollOnce() {
	pm.mu.Lock()
	if len(pm.tracked) == 0 {
		pm.mu.Unlock()
		return
	}
	snapshot := make(map[string]trackedProc, len(pm.tracked))
	for k, v := range pm.tracked {
		snapshot[k] = *v
	}
	pm.mu.Unlock()

	livePIDs := queryWindowsPIDs()
	portsByPID := queryListeningPortsByPID()

	var events []model.RunningProcess
	pm.mu.Lock()
	for pidKey, snap := range snapshot {
		cur, stillTracked := pm.tracked[pidKey]
		if !stillTracked {
			continue
		}
		if len(livePIDs) > 0 && !livePIDs[snap.pid] {
			delete(pm.tracked, pidKey)
			events = append(events, model.RunningProcess{ProjectID: pidKey, PID: 0, Ports: []int{}})
			continue
		}
		newPorts := portsByPID[snap.pid]
		if !intSlicesEqual(cur.ports, newPorts) {
			cur.ports = append([]int(nil), newPorts...)
			events = append(events, model.RunningProcess{
				ProjectID: cur.projectID,
				PID:       cur.pid,
				Ports:     append([]int(nil), cur.ports...),
				StartedAt: cur.startedAt,
				Command:   cur.command,
			})
		}
	}
	pm.mu.Unlock()

	if pm.onDelta != nil {
		for _, ev := range events {
			pm.onDelta(ev)
		}
	}
}

func queryWindowsPIDs() map[int]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "tasklist", "/FO", "CSV", "/NH")
	hideConsoleWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	pids := make(map[int]bool)
	for _, line := range bytes.Split(out, []byte("\n")) {
		fields := strings.Split(strings.TrimSpace(string(line)), ",")
		if len(fields) < 2 {
			continue
		}
		pidStr := strings.Trim(fields[1], `" `)
		if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
			pids[pid] = true
		}
	}
	return pids
}

func queryListeningPortsByPID() map[int][]int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "netstat", "-ano", "-p", "tcp")
	hideConsoleWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return ParseNetstatListeningPorts(string(out))
}

// ParseNetstatListeningPorts extracts listening TCP ports mapped by PID from `netstat -ano -p tcp` output.
func ParseNetstatListeningPorts(raw string) map[int][]int {
	out := make(map[int][]int)
	seen := make(map[string]bool)
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if !strings.EqualFold(fields[0], "TCP") || !strings.EqualFold(fields[3], "LISTENING") {
			continue
		}
		m := reListenAddrPort.FindStringSubmatch(fields[1])
		if len(m) != 2 {
			continue
		}
		port, err := strconv.Atoi(m[1])
		if err != nil || port <= 0 {
			continue
		}
		pid, err := strconv.Atoi(fields[4])
		if err != nil || pid <= 0 {
			continue
		}
		key := fmt.Sprintf("%d:%d", pid, port)
		if seen[key] {
			continue
		}
		seen[key] = true
		out[pid] = append(out[pid], port)
	}
	for pid := range out {
		sort.Ints(out[pid])
	}
	return out
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
