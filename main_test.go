package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestInitialModel(t *testing.T) {
	m := initialModel()

	if len(m.tasks) != 20 {
		t.Errorf("expected 20 tasks, got %d", len(m.tasks))
	}
	if len(m.workers) != 4 {
		t.Errorf("expected 4 workers, got %d", len(m.workers))
	}
	if m.view != viewDashboard {
		t.Errorf("expected dashboard view, got %d", m.view)
	}
	if m.filter != FilterAll {
		t.Errorf("expected FilterAll, got %d", m.filter)
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	if m.stats.TotalTasks != 20 {
		t.Errorf("expected total tasks 20, got %d", m.stats.TotalTasks)
	}
}

func TestTaskStatusString(t *testing.T) {
	tests := []struct {
		status TaskStatus
		want   string
	}{
		{StatusPending, "PENDING"},
		{StatusRunning, "RUNNING"},
		{StatusCompleted, "COMPLETED"},
		{StatusFailed, "FAILED"},
		{TaskStatus(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("TaskStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestWorkerStatusString(t *testing.T) {
	tests := []struct {
		status WorkerStatus
		want   string
	}{
		{WorkerIdle, "IDLE"},
		{WorkerBusy, "BUSY"},
		{WorkerOffline, "OFFLINE"},
		{WorkerStatus(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("WorkerStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestStatusFilterString(t *testing.T) {
	tests := []struct {
		filter StatusFilter
		want   string
	}{
		{FilterAll, "ALL"},
		{FilterPending, "PENDING"},
		{FilterRunning, "RUNNING"},
		{FilterCompleted, "COMPLETED"},
		{FilterFailed, "FAILED"},
	}
	for _, tt := range tests {
		if got := tt.filter.String(); got != tt.want {
			t.Errorf("StatusFilter(%d).String() = %q, want %q", tt.filter, got, tt.want)
		}
	}
}

func TestStatusFilterMatches(t *testing.T) {
	if !FilterAll.matches(StatusPending) {
		t.Error("FilterAll should match StatusPending")
	}
	if !FilterAll.matches(StatusRunning) {
		t.Error("FilterAll should match StatusRunning")
	}
	if !FilterPending.matches(StatusPending) {
		t.Error("FilterPending should match StatusPending")
	}
	if FilterPending.matches(StatusRunning) {
		t.Error("FilterPending should not match StatusRunning")
	}
	if !FilterRunning.matches(StatusRunning) {
		t.Error("FilterRunning should match StatusRunning")
	}
	if FilterRunning.matches(StatusCompleted) {
		t.Error("FilterRunning should not match StatusCompleted")
	}
	if !FilterCompleted.matches(StatusCompleted) {
		t.Error("FilterCompleted should match StatusCompleted")
	}
	if !FilterFailed.matches(StatusFailed) {
		t.Error("FilterFailed should match StatusFailed")
	}
	if FilterFailed.matches(StatusPending) {
		t.Error("FilterFailed should not match StatusPending")
	}
}

func TestFilteredTasks(t *testing.T) {
	m := initialModel()

	// FilterAll returns all tasks
	all := m.filteredTasks()
	if len(all) != 20 {
		t.Errorf("FilterAll: expected 20 tasks, got %d", len(all))
	}

	// Count statuses in mock data
	counts := map[TaskStatus]int{}
	for _, task := range m.tasks {
		counts[task.Status]++
	}

	// Test each filter
	filters := []struct {
		filter StatusFilter
		status TaskStatus
	}{
		{FilterPending, StatusPending},
		{FilterRunning, StatusRunning},
		{FilterCompleted, StatusCompleted},
		{FilterFailed, StatusFailed},
	}
	for _, f := range filters {
		m.filter = f.filter
		filtered := m.filteredTasks()
		if len(filtered) != counts[f.status] {
			t.Errorf("filter %s: expected %d tasks, got %d", f.filter, counts[f.status], len(filtered))
		}
		for _, task := range filtered {
			if task.Status != f.status {
				t.Errorf("filter %s: got task with status %s", f.filter, task.Status)
			}
		}
	}
}

func TestFilteredTasksSortByPriority(t *testing.T) {
	m := initialModel()
	m.sortByPrio = true
	filtered := m.filteredTasks()

	for i := 1; i < len(filtered); i++ {
		if filtered[i].Priority > filtered[i-1].Priority {
			t.Errorf("sort by priority: task %d (prio %d) > task %d (prio %d)",
				i, filtered[i].Priority, i-1, filtered[i-1].Priority)
		}
	}
}

func TestComputeStats(t *testing.T) {
	now := time.Now()
	tasks := []Task{
		{Status: StatusPending},
		{Status: StatusPending},
		{Status: StatusRunning},
		{Status: StatusCompleted, StartedAt: now.Add(-2 * time.Minute), CompletedAt: now.Add(-1 * time.Minute)},
		{Status: StatusCompleted, StartedAt: now.Add(-4 * time.Minute), CompletedAt: now.Add(-1 * time.Minute)},
		{Status: StatusFailed},
	}

	stats := computeStats(tasks)

	if stats.TotalTasks != 6 {
		t.Errorf("TotalTasks: expected 6, got %d", stats.TotalTasks)
	}
	if stats.QueueDepth != 2 {
		t.Errorf("QueueDepth: expected 2, got %d", stats.QueueDepth)
	}
	if stats.RunningTasks != 1 {
		t.Errorf("RunningTasks: expected 1, got %d", stats.RunningTasks)
	}
	if stats.CompletedTasks != 2 {
		t.Errorf("CompletedTasks: expected 2, got %d", stats.CompletedTasks)
	}
	if stats.FailedTasks != 1 {
		t.Errorf("FailedTasks: expected 1, got %d", stats.FailedTasks)
	}
	// Average latency: (1min + 3min) / 2 = 2min
	expectedLatency := 2 * time.Minute
	if stats.AvgLatency != expectedLatency {
		t.Errorf("AvgLatency: expected %s, got %s", expectedLatency, stats.AvgLatency)
	}
}

func TestCursorNavigation(t *testing.T) {
	m := initialModel()

	// Move down
	updated, _ := m.Update(keyMsg("j"))
	m = updated.(model)
	if m.cursor != 1 {
		t.Errorf("after j: expected cursor 1, got %d", m.cursor)
	}

	// Move down again
	updated, _ = m.Update(keyMsg("down"))
	m = updated.(model)
	if m.cursor != 2 {
		t.Errorf("after down: expected cursor 2, got %d", m.cursor)
	}

	// Move up
	updated, _ = m.Update(keyMsg("k"))
	m = updated.(model)
	if m.cursor != 1 {
		t.Errorf("after k: expected cursor 1, got %d", m.cursor)
	}

	// Move up again
	updated, _ = m.Update(keyMsg("up"))
	m = updated.(model)
	if m.cursor != 0 {
		t.Errorf("after up: expected cursor 0, got %d", m.cursor)
	}

	// Can't go above 0
	updated, _ = m.Update(keyMsg("k"))
	m = updated.(model)
	if m.cursor != 0 {
		t.Errorf("at top, after k: expected cursor 0, got %d", m.cursor)
	}
}

func TestCursorBoundsAtBottom(t *testing.T) {
	m := initialModel()

	// Move cursor to last item
	for i := 0; i < 25; i++ {
		updated, _ := m.Update(keyMsg("j"))
		m = updated.(model)
	}

	filtered := m.filteredTasks()
	if m.cursor != len(filtered)-1 {
		t.Errorf("expected cursor at %d, got %d", len(filtered)-1, m.cursor)
	}
}

func TestFilterCycling(t *testing.T) {
	m := initialModel()

	// Cycle through filters
	expected := []StatusFilter{FilterPending, FilterRunning, FilterCompleted, FilterFailed, FilterAll}
	for _, want := range expected {
		updated, _ := m.Update(keyMsg("f"))
		m = updated.(model)
		if m.filter != want {
			t.Errorf("expected filter %s, got %s", want, m.filter)
		}
		if m.cursor != 0 {
			t.Errorf("filter change should reset cursor to 0, got %d", m.cursor)
		}
	}
}

func TestSortToggle(t *testing.T) {
	m := initialModel()

	if m.sortByPrio {
		t.Error("sortByPrio should be false initially")
	}

	updated, _ := m.Update(keyMsg("s"))
	m = updated.(model)
	if !m.sortByPrio {
		t.Error("after s: sortByPrio should be true")
	}

	updated, _ = m.Update(keyMsg("s"))
	m = updated.(model)
	if m.sortByPrio {
		t.Error("after second s: sortByPrio should be false")
	}
}

func TestViewDetail(t *testing.T) {
	m := initialModel()

	// Press enter to view detail
	updated, _ := m.Update(keyMsg("enter"))
	m = updated.(model)
	if m.view != viewDetail {
		t.Errorf("after enter: expected viewDetail, got %d", m.view)
	}
	if m.selectedTask == nil {
		t.Fatal("selectedTask should not be nil")
	}
	if m.selectedTask.ID != m.tasks[0].ID {
		t.Errorf("selectedTask ID: expected %s, got %s", m.tasks[0].ID, m.selectedTask.ID)
	}

	// Press esc to go back
	updated, _ = m.Update(keyMsg("esc"))
	m = updated.(model)
	if m.view != viewDashboard {
		t.Errorf("after esc: expected viewDashboard, got %d", m.view)
	}
	if m.selectedTask != nil {
		t.Error("selectedTask should be nil after esc")
	}
}

func TestViewHelp(t *testing.T) {
	m := initialModel()

	// Press ? to view help
	updated, _ := m.Update(keyMsg("?"))
	m = updated.(model)
	if m.view != viewHelp {
		t.Errorf("after ?: expected viewHelp, got %d", m.view)
	}

	// Press ? again to go back
	updated, _ = m.Update(keyMsg("?"))
	m = updated.(model)
	if m.view != viewDashboard {
		t.Errorf("after second ?: expected viewDashboard, got %d", m.view)
	}
}

func TestTickUpdatesProgress(t *testing.T) {
	m := initialModel()

	// Find a running task
	var runningIdx int
	found := false
	for i, task := range m.tasks {
		if task.Status == StatusRunning {
			runningIdx = i
			found = true
			break
		}
	}
	if !found {
		t.Skip("no running tasks in mock data")
	}

	origProgress := m.tasks[runningIdx].Progress

	// Send tick
	updated, _ := m.Update(tickMsg(time.Now()))
	m = updated.(model)

	if m.tasks[runningIdx].Status == StatusRunning {
		if m.tasks[runningIdx].Progress <= origProgress {
			t.Errorf("tick should increase progress: was %d, now %d",
				origProgress, m.tasks[runningIdx].Progress)
		}
	}
	// If task completed (progress >= 100), that's also valid
}

func TestPriorityLabel(t *testing.T) {
	tests := []struct {
		prio int
		want string
	}{
		{1, "MIN"},
		{2, "LOW"},
		{3, "MED"},
		{4, "HIGH"},
		{5, "CRIT"},
		{0, "?"},
	}
	for _, tt := range tests {
		if got := priorityLabel(tt.prio); got != tt.want {
			t.Errorf("priorityLabel(%d) = %q, want %q", tt.prio, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate short: got %q", got)
	}
	if got := truncate("a-very-long-string-here", 10); got != "a-very-l.." {
		t.Errorf("truncate long: got %q", got)
	}
	if got := truncate("exact", 5); got != "exact" {
		t.Errorf("truncate exact: got %q", got)
	}
}

func TestOrDash(t *testing.T) {
	if got := orDash(""); got != "-" {
		t.Errorf("orDash empty: got %q", got)
	}
	if got := orDash("value"); got != "value" {
		t.Errorf("orDash value: got %q", got)
	}
}

func TestFormatDuration(t *testing.T) {
	if got := formatDuration(0); got != "-" {
		t.Errorf("formatDuration(0): got %q", got)
	}
	if got := formatDuration(72 * time.Hour); got != "72h00m" {
		t.Errorf("formatDuration(72h): got %q", got)
	}
	if got := formatDuration(25*time.Hour + 30*time.Minute); got != "25h30m" {
		t.Errorf("formatDuration(25h30m): got %q", got)
	}
}

func TestGenerateMockWorkers(t *testing.T) {
	tasks := generateMockTasks()
	workers := generateMockWorkers(tasks)

	if len(workers) != 4 {
		t.Errorf("expected 4 workers, got %d", len(workers))
	}

	// Verify busy workers have current tasks assigned (if running tasks exist for them)
	for _, w := range workers {
		if w.Status == WorkerBusy && w.CurrentTaskID != "" {
			found := false
			for _, task := range tasks {
				if task.ID == w.CurrentTaskID && task.Status == StatusRunning {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("worker %s has currentTask %s but no matching running task",
					w.ID, w.CurrentTaskID)
			}
		}
	}
}

func TestDashboardViewContainsAllSections(t *testing.T) {
	m := initialModel()
	view := m.View()

	sections := []string{
		"Distributed Task Scheduler",
		"Task Queue",
		"Workers",
		"Scheduler Stats",
		"j/k: navigate",
	}
	for _, s := range sections {
		if !strings.Contains(view, s) {
			t.Errorf("dashboard view missing section: %q", s)
		}
	}
}

func TestDetailViewContent(t *testing.T) {
	m := initialModel()

	// Select first task
	updated, _ := m.Update(keyMsg("enter"))
	m = updated.(model)

	view := m.View()
	if !strings.Contains(view, "Task Detail") {
		t.Error("detail view missing 'Task Detail' header")
	}
	if !strings.Contains(view, m.selectedTask.ID) {
		t.Errorf("detail view missing task ID %s", m.selectedTask.ID)
	}
	if !strings.Contains(view, m.selectedTask.Name) {
		t.Errorf("detail view missing task name %s", m.selectedTask.Name)
	}
}

func TestHelpViewContent(t *testing.T) {
	m := initialModel()

	updated, _ := m.Update(keyMsg("?"))
	m = updated.(model)

	view := m.View()
	if !strings.Contains(view, "Help") {
		t.Error("help view missing 'Help' header")
	}
	if !strings.Contains(view, "Keybindings") {
		t.Error("help view missing 'Keybindings'")
	}
}

func TestQuitFromDashboard(t *testing.T) {
	m := initialModel()
	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Error("q should return a quit command")
	}
}

func TestQuitFromDetail(t *testing.T) {
	m := initialModel()
	m.view = viewDetail
	m.selectedTask = &m.tasks[0]

	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Error("q from detail should return a quit command")
	}
}

func TestQuitFromHelp(t *testing.T) {
	m := initialModel()
	m.view = viewHelp

	_, cmd := m.Update(keyMsg("q"))
	if cmd == nil {
		t.Error("q from help should return a quit command")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := initialModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	m = updated.(model)

	if m.width != 200 {
		t.Errorf("expected width 200, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("expected height 50, got %d", m.height)
	}
}

func TestFilterResetsCursor(t *testing.T) {
	m := initialModel()

	// Move cursor down
	for i := 0; i < 5; i++ {
		updated, _ := m.Update(keyMsg("j"))
		m = updated.(model)
	}
	if m.cursor != 5 {
		t.Fatalf("expected cursor at 5, got %d", m.cursor)
	}

	// Change filter resets cursor
	updated, _ := m.Update(keyMsg("f"))
	m = updated.(model)
	if m.cursor != 0 {
		t.Errorf("filter change should reset cursor to 0, got %d", m.cursor)
	}
}

func TestMockDataDeterministic(t *testing.T) {
	tasks1 := generateMockTasks()
	tasks2 := generateMockTasks()

	if len(tasks1) != len(tasks2) {
		t.Fatal("mock data should be deterministic")
	}
	for i := range tasks1 {
		if tasks1[i].ID != tasks2[i].ID || tasks1[i].Status != tasks2[i].Status ||
			tasks1[i].Priority != tasks2[i].Priority {
			t.Errorf("task %d differs between runs", i)
		}
	}
}
