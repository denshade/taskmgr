package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseDeadline(t *testing.T) {
	loc := time.Local
	want := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, 0, 0, 0, 0, loc)
	}

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{"iso", "2026-04-04", want(2026, time.April, 4), false},
		{"iso trimmed", "  2026-04-04  ", want(2026, time.April, 4), false},
		{"dd/mm/yyyy padded", "04/04/2026", want(2026, time.April, 4), false},
		{"d/m/yyyy", "4/4/2026", want(2026, time.April, 4), false},
		{"d/mm/yyyy", "4/04/2026", want(2026, time.April, 4), false},
		{"dd/m/yyyy", "04/4/2026", want(2026, time.April, 4), false},
		{"empty", "", time.Time{}, true},
		{"whitespace only", "   ", time.Time{}, true},
		{"invalid", "not-a-date", time.Time{}, true},
		{"wrong separator iso", "2026/04/04", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDeadline(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestDeadlineFromInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty", "", "", false},
		{"whitespace", "  \t  ", "", false},
		{"iso", "2026-04-04", "2026-04-04", false},
		{"invalid", "nope", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := deadlineFromInput(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestLoadTasks(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "none.json")
		tasks, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if tasks != nil {
			t.Fatalf("got %#v want nil", tasks)
		}
	})

	t.Run("empty file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := os.WriteFile(path, []byte(" \n\t "), 0644); err != nil {
			t.Fatal(err)
		}
		tasks, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if tasks != nil {
			t.Fatalf("got %#v want nil", tasks)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := loadTasks(path)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestSaveTasksLoadTasksRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	want := []Task{
		{Description: "a", Steps: 2, Deadline: "2026-01-01"},
		{Description: "b", Steps: 1, Deadline: "2026-12-31"},
		{Description: "no due", Steps: 3, Deadline: ""},
	}
	if err := saveTasks(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadTasks(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestTaskJSONOmitsEmptyDeadline(t *testing.T) {
	raw, err := json.Marshal(Task{Description: "x", Steps: 1, Deadline: ""})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "deadline") {
		t.Fatalf("expected deadline omitted, got %s", raw)
	}
}

func TestReadLine(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("hello world\n"))
	got, err := readLine(r)
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestParseAddArgs(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "other-tasks.json")

	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{"default", nil, defaultTasksFile, false},
		{"default empty slice", []string{}, defaultTasksFile, false},
		{"f short", []string{"-f", custom}, custom, false},
		{"f equals", []string{"-f=" + custom}, custom, false},
		{"file long", []string{"-file", custom}, custom, false},
		{"file equals", []string{"-file=" + custom}, custom, false},
		{"unexpected arg", []string{"extra"}, "", true},
		{"unknown flag", []string{"-nope"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAddArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestParseDeleteArgs(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "other-tasks.json")

	tests := []struct {
		name      string
		args      []string
		wantPath  string
		wantIndex int
		wantErr   bool
	}{
		{"index only", []string{"1"}, defaultTasksFile, 1, false},
		{"index before -f", []string{"2", "-f", custom}, custom, 2, false},
		{"-f before index", []string{"-f", custom, "3"}, custom, 3, false},
		{"-file before index", []string{"-file", custom, "1"}, custom, 1, false},
		{"-f= equals", []string{"-f=" + custom, "1"}, custom, 1, false},
		{"-file= equals", []string{"-file=" + custom, "2"}, custom, 2, false},
		{"missing index", nil, "", 0, true},
		{"too many positionals", []string{"1", "2"}, "", 0, true},
		{"invalid index", []string{"0"}, "", 0, true},
		{"non-numeric index", []string{"x"}, "", 0, true},
		{"-f without value", []string{"-f"}, "", 0, true},
		{"unknown flag", []string{"-nope", "1"}, "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotIdx, err := parseDeleteArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path got %q want %q", gotPath, tt.wantPath)
			}
			if gotIdx != tt.wantIndex {
				t.Fatalf("index got %d want %d", gotIdx, tt.wantIndex)
			}
		})
	}
}

func TestCmdDelete(t *testing.T) {
	t.Run("removes task and rewrites file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		start := []Task{
			{Description: "first", Steps: 1},
			{Description: "second", Steps: 2},
			{Description: "third", Steps: 3},
		}
		if err := saveTasks(path, start); err != nil {
			t.Fatal(err)
		}

		if err := cmdDelete([]string{"-f", path, "2"}); err != nil {
			t.Fatal(err)
		}

		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		want := []Task{
			{Description: "first", Steps: 1},
			{Description: "third", Steps: 3},
		}
		if len(got) != len(want) {
			t.Fatalf("len got %d want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("idx %d: got %+v want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("empty file error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := cmdDelete([]string{"-f", path, "1"}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("out of range", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := saveTasks(path, []Task{{Description: "only", Steps: 1}}); err != nil {
			t.Fatal(err)
		}
		if err := cmdDelete([]string{"-f", path, "9"}); err == nil {
			t.Fatal("expected error")
		}
	})
}
