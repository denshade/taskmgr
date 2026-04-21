package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHistoryCSVPath(t *testing.T) {
	got := historyCSVPath(filepath.Join("data", "tasks.json"))
	want := filepath.Join("data", "history.csv")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAppendTaskEditHistory(t *testing.T) {
	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.json")
	ts := time.Date(2026, 4, 4, 12, 30, 45, 0, time.UTC)

	if err := appendTaskEditHistory(tasksPath, ts, "Bananen", 3, 10); err != nil {
		t.Fatal(err)
	}
	rows := readHistoryCSV(t, filepath.Join(dir, "history.csv"))
	if len(rows) != 2 {
		t.Fatalf("want 2 rows (header + data), got %d: %v", len(rows), rows)
	}
	if got := strings.Join(rows[0], ","); got != "timestamp,task,progress,steps" {
		t.Fatalf("header: %q", got)
	}
	if rows[1][0] != ts.Format(time.RFC3339) || rows[1][1] != "Bananen" || rows[1][2] != "3" || rows[1][3] != "10" {
		t.Fatalf("data row: %v", rows[1])
	}

	ts2 := ts.Add(time.Hour)
	if err := appendTaskEditHistory(tasksPath, ts2, "Bananen", 5, 10); err != nil {
		t.Fatal(err)
	}
	rows = readHistoryCSV(t, filepath.Join(dir, "history.csv"))
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	if rows[2][2] != "5" {
		t.Fatalf("second progress: %v", rows[2])
	}
}

func readHistoryCSV(t *testing.T, path string) [][]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r := csv.NewReader(strings.NewReader(string(data)))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestEditTaskWritesHistoryCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	if err := saveTasks(path, []Task{{Description: "x", Steps: 3, Progress: 1, Deadline: "2026-01-01"}}); err != nil {
		t.Fatal(err)
	}
	in := strings.NewReader("\n2\n\n\n\n")
	if err := editTask(in, path, 1); err != nil {
		t.Fatal(err)
	}
	hpath := filepath.Join(dir, "history.csv")
	rows := readHistoryCSV(t, hpath)
	if len(rows) < 2 {
		t.Fatalf("expected header + row, got %v", rows)
	}
	last := rows[len(rows)-1]
	if last[1] != "x" || last[2] != "2" || last[3] != "3" {
		t.Fatalf("last row: %v", last)
	}
}
