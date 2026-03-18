package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	headerStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	selectedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statsStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("79"))
	pendingStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("227"))
	runningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	completedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	failedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	idleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	busyStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	offlineStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// TaskStatus represents the state of a task.
type TaskStatus int

const (
	StatusPending TaskStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
)

func (s TaskStatus) String() string {
	switch s {
	case StatusPending:
		return "PENDING"
	case StatusRunning:
		return "RUNNING"
	case StatusCompleted:
		return "COMPLETED"
	case StatusFailed:
		return "FAILED"
	}
	return "UNKNOWN"
}

// WorkerStatus represents the state of a worker.
type WorkerStatus int

const (
	WorkerIdle WorkerStatus = iota
	WorkerBusy
	WorkerOffline
)

func (s WorkerStatus) String() string {
	switch s {
	case WorkerIdle:
		return "IDLE"
	case WorkerBusy:
		return "BUSY"
	case WorkerOffline:
		return "OFFLINE"
	}
	return "UNKNOWN"
}

// Task represents a computational job.
type Task struct {
	ID          string
	Name        string
	Status      TaskStatus
	Priority    int // 1 (low) to 5 (critical)
	WorkerID    string
	SubmittedAt time.Time
	StartedAt   time.Time
	CompletedAt time.Time
	Progress    int // 0-100
	Error       string
}

// Worker represents a cluster node.
type Worker struct {
	ID             string
	Status         WorkerStatus
	CurrentTaskID  string
	Uptime         time.Duration
	TasksCompleted int
}

// SchedulerStats holds aggregate metrics.
type SchedulerStats struct {
	Throughput     float64
	AvgLatency     time.Duration
	QueueDepth     int
	TotalTasks     int
	CompletedTasks int
	FailedTasks    int
	RunningTasks   int
}

type viewMode int

const (
	viewDashboard viewMode = iota
	viewDetail
	viewHelp
)

// StatusFilter for task filtering.
type StatusFilter int

const (
	FilterAll StatusFilter = iota
	FilterPending
	FilterRunning
	FilterCompleted
	FilterFailed
)

func (f StatusFilter) String() string {
	switch f {
	case FilterPending:
		return "PENDING"
	case FilterRunning:
		return "RUNNING"
	case FilterCompleted:
		return "COMPLETED"
	case FilterFailed:
		return "FAILED"
	}
	return "ALL"
}

func (f StatusFilter) matches(s TaskStatus) bool {
	switch f {
	case FilterPending:
		return s == StatusPending
	case FilterRunning:
		return s == StatusRunning
	case FilterCompleted:
		return s == StatusCompleted
	case FilterFailed:
		return s == StatusFailed
	}
	return true
}

// tickMsg signals a periodic refresh.
type tickMsg time.Time

// model is the Bubble Tea model.
type model struct {
	tasks        []Task
	workers      []Worker
	stats        SchedulerStats
	cursor       int
	view         viewMode
	filter       StatusFilter
	sortByPrio   bool
	selectedTask *Task
	width        int
	height       int
}

func generateMockTasks() []Task {
	now := time.Now()
	names := []string{
		"data-ingestion-batch", "model-training-v2", "feature-extraction",
		"report-generation", "index-rebuild", "log-aggregation",
		"backup-snapshots", "cache-warmup", "schema-migration",
		"metric-rollup", "image-processing", "video-transcode",
		"email-dispatch", "fraud-detection", "recommendation-upd",
		"search-reindex", "audit-compliance", "data-export",
		"pipeline-cleanup", "health-check-sweep",
	}
	workerIDs := []string{"worker-01", "worker-02", "worker-03", "worker-04"}
	rng := rand.New(rand.NewSource(42))

	tasks := make([]Task, 20)
	for i := 0; i < 20; i++ {
		status := TaskStatus(rng.Intn(4))
		priority := rng.Intn(5) + 1
		t := Task{
			ID:          fmt.Sprintf("task-%03d", i+1),
			Name:        names[i],
			Status:      status,
			Priority:    priority,
			SubmittedAt: now.Add(-time.Duration(rng.Intn(3600)) * time.Second),
		}
		switch status {
		case StatusRunning:
			t.WorkerID = workerIDs[rng.Intn(4)]
			t.StartedAt = now.Add(-time.Duration(rng.Intn(600)) * time.Second)
			t.Progress = rng.Intn(90) + 10
		case StatusCompleted:
			t.WorkerID = workerIDs[rng.Intn(4)]
			t.StartedAt = now.Add(-time.Duration(rng.Intn(3600)) * time.Second)
			t.CompletedAt = t.StartedAt.Add(time.Duration(rng.Intn(300)+1) * time.Second)
			t.Progress = 100
		case StatusFailed:
			t.WorkerID = workerIDs[rng.Intn(4)]
			t.StartedAt = now.Add(-time.Duration(rng.Intn(3600)) * time.Second)
			t.CompletedAt = t.StartedAt.Add(time.Duration(rng.Intn(120)+1) * time.Second)
			t.Error = "exit code 1: out of memory"
		}
		tasks[i] = t
	}
	return tasks
}

func generateMockWorkers(tasks []Task) []Worker {
	workers := []Worker{
		{ID: "worker-01", Status: WorkerBusy, Uptime: 72 * time.Hour, TasksCompleted: 142},
		{ID: "worker-02", Status: WorkerBusy, Uptime: 48 * time.Hour, TasksCompleted: 98},
		{ID: "worker-03", Status: WorkerIdle, Uptime: 24 * time.Hour, TasksCompleted: 67},
		{ID: "worker-04", Status: WorkerOffline, Uptime: 0, TasksCompleted: 53},
	}
	for i := range workers {
		if workers[i].Status == WorkerBusy {
			for _, t := range tasks {
				if t.Status == StatusRunning && t.WorkerID == workers[i].ID {
					workers[i].CurrentTaskID = t.ID
					break
				}
			}
		}
	}
	return workers
}

func computeStats(tasks []Task) SchedulerStats {
	stats := SchedulerStats{TotalTasks: len(tasks)}
	var totalLatency time.Duration
	completedCount := 0

	for _, t := range tasks {
		switch t.Status {
		case StatusPending:
			stats.QueueDepth++
		case StatusRunning:
			stats.RunningTasks++
		case StatusCompleted:
			stats.CompletedTasks++
			if !t.CompletedAt.IsZero() && !t.StartedAt.IsZero() {
				totalLatency += t.CompletedAt.Sub(t.StartedAt)
				completedCount++
			}
		case StatusFailed:
			stats.FailedTasks++
		}
	}
	if completedCount > 0 {
		stats.AvgLatency = totalLatency / time.Duration(completedCount)
		stats.Throughput = float64(completedCount) / 60.0 // tasks per second (over 1 min)
	}
	return stats
}

func initialModel() model {
	tasks := generateMockTasks()
	workers := generateMockWorkers(tasks)
	stats := computeStats(tasks)
	return model{
		tasks:   tasks,
		workers: workers,
		stats:   stats,
		view:    viewDashboard,
		filter:  FilterAll,
		width:   120,
		height:  40,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) filteredTasks() []Task {
	var filtered []Task
	for _, t := range m.tasks {
		if m.filter.matches(t.Status) {
			filtered = append(filtered, t)
		}
	}
	if m.sortByPrio {
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].Priority > filtered[j].Priority
		})
	}
	return filtered
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		for i := range m.tasks {
			if m.tasks[i].Status == StatusRunning {
				m.tasks[i].Progress += rand.Intn(5) + 1
				if m.tasks[i].Progress >= 100 {
					m.tasks[i].Progress = 100
					m.tasks[i].Status = StatusCompleted
					m.tasks[i].CompletedAt = time.Now()
				}
			}
		}
		m.stats = computeStats(m.tasks)
		m.workers = generateMockWorkers(m.tasks)
		return m, tickCmd()

	case tea.KeyMsg:
		switch m.view {
		case viewDashboard:
			return m.updateDashboard(msg)
		case viewDetail:
			return m.updateDetail(msg)
		case viewHelp:
			return m.updateHelp(msg)
		}
	}
	return m, nil
}

func (m model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredTasks()
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(filtered)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if len(filtered) > 0 && m.cursor < len(filtered) {
			task := filtered[m.cursor]
			m.selectedTask = &task
			m.view = viewDetail
		}
	case "f":
		m.filter = (m.filter + 1) % 5
		m.cursor = 0
	case "s":
		m.sortByPrio = !m.sortByPrio
	case "?":
		m.view = viewHelp
	}
	return m, nil
}

func (m model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		m.view = viewDashboard
		m.selectedTask = nil
	}
	return m, nil
}

func (m model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace", "?":
		m.view = viewDashboard
	}
	return m, nil
}

func (m model) View() string {
	switch m.view {
	case viewDetail:
		return m.viewDetail()
	case viewHelp:
		return m.viewHelp()
	default:
		return m.viewDashboard()
	}
}

func statusStyle(s TaskStatus) lipgloss.Style {
	switch s {
	case StatusPending:
		return pendingStyle
	case StatusRunning:
		return runningStyle
	case StatusCompleted:
		return completedStyle
	case StatusFailed:
		return failedStyle
	}
	return lipgloss.NewStyle()
}

func workerStyle(s WorkerStatus) lipgloss.Style {
	switch s {
	case WorkerIdle:
		return idleStyle
	case WorkerBusy:
		return busyStyle
	case WorkerOffline:
		return offlineStyle
	}
	return lipgloss.NewStyle()
}

func priorityLabel(p int) string {
	switch p {
	case 5:
		return "CRIT"
	case 4:
		return "HIGH"
	case 3:
		return "MED"
	case 2:
		return "LOW"
	case 1:
		return "MIN"
	}
	return "?"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-2] + ".."
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", h, m)
}

func (m model) viewDashboard() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Distributed Task Scheduler"))
	b.WriteString("\n\n")

	// Task queue header
	filtered := m.filteredTasks()
	sortLabel := "default"
	if m.sortByPrio {
		sortLabel = "priority"
	}
	header := fmt.Sprintf("Task Queue [filter: %s] [sort: %s] (%d tasks)",
		m.filter.String(), sortLabel, len(filtered))
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("  %-10s %-24s %-11s %-6s %-12s %s\n",
		"ID", "NAME", "STATUS", "PRIO", "WORKER", "PROGRESS"))
	b.WriteString(strings.Repeat("─", 78))
	b.WriteString("\n")

	for i, t := range filtered {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		status := statusStyle(t.Status).Render(fmt.Sprintf("%-9s", t.Status))
		prio := priorityLabel(t.Priority)
		worker := orDash(t.WorkerID)
		progress := ""
		if t.Status == StatusRunning {
			progress = fmt.Sprintf("%d%%", t.Progress)
		} else if t.Status == StatusCompleted {
			progress = "done"
		} else if t.Status == StatusFailed {
			progress = "err"
		}

		line := fmt.Sprintf("%s%-10s %-24s %s %-6s %-12s %s",
			cursor, t.ID, truncate(t.Name, 23), status, prio, worker, progress)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Workers
	b.WriteString(headerStyle.Render("Workers"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %-12s %-9s %-14s %-14s %s\n",
		"ID", "STATUS", "CURRENT TASK", "UPTIME", "COMPLETED"))
	b.WriteString(strings.Repeat("─", 62))
	b.WriteString("\n")

	for _, w := range m.workers {
		status := workerStyle(w.Status).Render(fmt.Sprintf("%-7s", w.Status))
		currentTask := orDash(w.CurrentTaskID)
		uptime := formatDuration(w.Uptime)
		b.WriteString(fmt.Sprintf("  %-12s %s %-14s %-14s %d\n",
			w.ID, status, currentTask, uptime, w.TasksCompleted))
	}

	b.WriteString("\n")

	// Stats
	b.WriteString(headerStyle.Render("Scheduler Stats"))
	b.WriteString("\n")
	b.WriteString(statsStyle.Render(fmt.Sprintf(
		"  Total: %d  Pending: %d  Running: %d  Completed: %d  Failed: %d\n"+
			"  Avg Latency: %s  Queue Depth: %d",
		m.stats.TotalTasks, m.stats.QueueDepth, m.stats.RunningTasks,
		m.stats.CompletedTasks, m.stats.FailedTasks,
		m.stats.AvgLatency.Round(time.Second), m.stats.QueueDepth)))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("j/k: navigate  enter: detail  f: filter  s: sort  ?: help  q: quit"))
	return b.String()
}

func (m model) viewDetail() string {
	if m.selectedTask == nil {
		return "No task selected"
	}
	t := m.selectedTask
	var b strings.Builder

	b.WriteString(titleStyle.Render("Task Detail"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  ID:          %s\n", t.ID))
	b.WriteString(fmt.Sprintf("  Name:        %s\n", t.Name))
	b.WriteString(fmt.Sprintf("  Status:      %s\n", statusStyle(t.Status).Render(t.Status.String())))
	b.WriteString(fmt.Sprintf("  Priority:    %d (%s)\n", t.Priority, priorityLabel(t.Priority)))
	b.WriteString(fmt.Sprintf("  Worker:      %s\n", orDash(t.WorkerID)))
	b.WriteString(fmt.Sprintf("  Submitted:   %s\n", t.SubmittedAt.Format(time.RFC3339)))
	if !t.StartedAt.IsZero() {
		b.WriteString(fmt.Sprintf("  Started:     %s\n", t.StartedAt.Format(time.RFC3339)))
	}
	if !t.CompletedAt.IsZero() {
		b.WriteString(fmt.Sprintf("  Completed:   %s\n", t.CompletedAt.Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("  Duration:    %s\n", t.CompletedAt.Sub(t.StartedAt).Round(time.Second)))
	}
	if t.Status == StatusRunning {
		b.WriteString(fmt.Sprintf("  Progress:    %d%%\n", t.Progress))
	}
	if t.Error != "" {
		b.WriteString(fmt.Sprintf("  Error:       %s\n", failedStyle.Render(t.Error)))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc: back  q: quit"))
	return b.String()
}

func (m model) viewHelp() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Help"))
	b.WriteString("\n\n")
	b.WriteString("  Keybindings:\n\n")
	b.WriteString("  j / down     Move cursor down\n")
	b.WriteString("  k / up       Move cursor up\n")
	b.WriteString("  enter        View task detail\n")
	b.WriteString("  f            Cycle status filter\n")
	b.WriteString("  s            Toggle sort by priority\n")
	b.WriteString("  ?            Toggle help\n")
	b.WriteString("  esc          Back to dashboard\n")
	b.WriteString("  q / ctrl+c   Quit\n")
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("esc/?: back  q: quit"))
	return b.String()
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
