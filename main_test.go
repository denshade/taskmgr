package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWinresIconUsesDeadlinePNG(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("winres", "winres.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		RT_GROUP_ICON map[string]map[string]string `json:"RT_GROUP_ICON"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	app, ok := v.RT_GROUP_ICON["APP"]
	if !ok {
		t.Fatal(`winres: missing RT_GROUP_ICON "APP"`)
	}
	const want = "../deadline.png"
	if got := app["0000"]; got != want {
		t.Fatalf("icon input: got %q, want %q", got, want)
	}
	pngPath := filepath.Join("winres", want)
	if _, err := os.Stat(pngPath); err != nil {
		t.Fatalf("icon file %q: %v", pngPath, err)
	}
}

func TestHelpText(t *testing.T) {
	s := helpText()
	if !strings.HasPrefix(s, "deadline ") {
		t.Fatalf("want title starting with app name, got first line %q", strings.SplitN(s, "\n", 2)[0])
	}
	if !strings.Contains(s, "deadline <command>") {
		t.Fatalf("usage line should name binary deadline, got substring check failed")
	}
	if strings.Contains(s, "taskmgr") {
		t.Fatal("help should not reference old app name taskmgr")
	}
}

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

func TestReadLine_EOFWithPartialLine(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("no newline"))
	got, err := readLine(r)
	if err != nil {
		t.Fatalf("got err %v want nil for partial line before EOF", err)
	}
	if got != "no newline" {
		t.Fatalf("got %q", got)
	}
	_, err = readLine(r)
	if err != io.EOF {
		t.Fatalf("second read: got %v want EOF", err)
	}
}

func TestParseTasksFilePathArgs(t *testing.T) {
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
			got, err := parseTasksFilePathArgs(tt.args)
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

func TestValidateProgress(t *testing.T) {
	tests := []struct {
		name     string
		progress int
		steps    int
		wantErr  bool
	}{
		{"zero steps zero progress", 0, 1, false},
		{"at cap", 5, 5, false},
		{"below cap", 2, 5, false},
		{"negative", -1, 5, true},
		{"above steps", 6, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProgress(tt.progress, tt.steps)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEditTask(t *testing.T) {
	t.Run("enter keeps all fields", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		start := []Task{
			{Description: "Bananas", Steps: 3, Progress: 1, Deadline: "2026-01-01"},
			{Description: "other", Steps: 1, Deadline: "2026-02-01"},
		}
		if err := saveTasks(path, start); err != nil {
			t.Fatal(err)
		}
		in := strings.NewReader("\n\n\n\n")
		if err := editTask(in, path, 1); err != nil {
			t.Fatal(err)
		}
		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if got[0] != start[0] || got[1] != start[1] {
			t.Fatalf("got %+v %+v want %+v %+v", got[0], got[1], start[0], start[1])
		}
	})

	t.Run("changes description and progress", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := saveTasks(path, []Task{{Description: "old", Steps: 5, Progress: 0, Deadline: "2026-03-01"}}); err != nil {
			t.Fatal(err)
		}
		in := strings.NewReader("new desc\n\n\n4\n")
		if err := editTask(in, path, 1); err != nil {
			t.Fatal(err)
		}
		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Description != "new desc" || got[0].Progress != 4 || got[0].Steps != 5 {
			t.Fatalf("got %+v", got[0])
		}
	})

	t.Run("progress equal to steps", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := saveTasks(path, []Task{{Description: "x", Steps: 2, Deadline: "2026-03-01"}}); err != nil {
			t.Fatal(err)
		}
		in := strings.NewReader("\n\n\n2\n")
		if err := editTask(in, path, 1); err != nil {
			t.Fatal(err)
		}
		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Progress != 2 {
			t.Fatalf("progress got %d", got[0].Progress)
		}
	})

	t.Run("empty file error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := editTask(strings.NewReader("\n\n\n\n"), path, 1); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("out of range index", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := saveTasks(path, []Task{{Description: "only", Steps: 1, Deadline: "2026-01-01"}}); err != nil {
			t.Fatal(err)
		}
		if err := editTask(strings.NewReader("\n\n\n\n"), path, 9); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("reducing steps requires valid progress", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := saveTasks(path, []Task{{Description: "x", Steps: 5, Progress: 4, Deadline: "2026-01-01"}}); err != nil {
			t.Fatal(err)
		}
		in := strings.NewReader("\n2\n\n\n2\n")
		if err := editTask(in, path, 1); err != nil {
			t.Fatal(err)
		}
		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Steps != 2 || got[0].Progress != 2 {
			t.Fatalf("got %+v", got[0])
		}
	})

	t.Run("rejects negative progress then accepts", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := saveTasks(path, []Task{{Description: "only", Steps: 3, Deadline: "2026-01-01"}}); err != nil {
			t.Fatal(err)
		}
		in := strings.NewReader("\n\n\n-1\n0\n")
		if err := editTask(in, path, 1); err != nil {
			t.Fatal(err)
		}
		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Progress != 0 {
			t.Fatalf("progress got %d", got[0].Progress)
		}
	})

	t.Run("rejects progress over steps then accepts", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := saveTasks(path, []Task{{Description: "only", Steps: 2, Deadline: "2026-01-01"}}); err != nil {
			t.Fatal(err)
		}
		in := strings.NewReader("\n\n\n9\n1\n")
		if err := editTask(in, path, 1); err != nil {
			t.Fatal(err)
		}
		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Progress != 1 {
			t.Fatalf("progress got %d", got[0].Progress)
		}
	})
}

func TestCmdEdit(t *testing.T) {
	t.Run("too many positionals fails before stdin", func(t *testing.T) {
		if err := cmdEdit([]string{"1", "2"}); err == nil {
			t.Fatal("expected error")
		}
	})
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

func TestWriteTaskList(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var buf bytes.Buffer
		writeTaskList(&buf, nil)
		if got := strings.TrimSpace(buf.String()); got != "No tasks." {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("one task with deadline", func(t *testing.T) {
		var buf bytes.Buffer
		writeTaskList(&buf, []Task{
			{Description: "Do thing", Progress: 1, Steps: 3, Deadline: "2026-04-06"},
		})
		want := strings.TrimSpace(`
Index: 1
Description: Do thing
Current progress: 1
#steps: 3
Deadline: 2026-04-06`)
		if got := strings.TrimSpace(buf.String()); got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("two tasks second has no deadline", func(t *testing.T) {
		var buf bytes.Buffer
		writeTaskList(&buf, []Task{
			{Description: "a", Progress: 0, Steps: 2, Deadline: "2026-01-01"},
			{Description: "b", Progress: 2, Steps: 2, Deadline: ""},
		})
		want := strings.TrimSpace(`
Index: 1
Description: a
Current progress: 0
#steps: 2
Deadline: 2026-01-01

Index: 2
Description: b
Current progress: 2
#steps: 2
Deadline: (none)`)
		if got := strings.TrimSpace(buf.String()); got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})
}

func TestWriteTasksFromPath(t *testing.T) {
	t.Run("prints tasks from file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		if err := saveTasks(path, []Task{
			{Description: "x", Progress: 0, Steps: 1, Deadline: "2026-05-01"},
		}); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := writeTasksFromPath(&buf, path); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, "Index: 1") || !strings.Contains(out, "Description: x") {
			t.Fatalf("output: %q", out)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := writeTasksFromPath(&bytes.Buffer{}, path); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCmdList(t *testing.T) {
	t.Run("invalid args", func(t *testing.T) {
		if err := cmdList([]string{"-nope"}); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestLatestDeadlineDay(t *testing.T) {
	loc := time.Local

	tests := []struct {
		name    string
		tasks   []Task
		want    time.Time
		wantErr bool
	}{
		{
			name:    "no deadlines",
			tasks:   []Task{{Description: "a", Steps: 1, Deadline: ""}},
			wantErr: true,
		},
		{
			name: "single",
			tasks: []Task{
				{Description: "a", Steps: 1, Deadline: "2026-04-06"},
			},
			want: time.Date(2026, 4, 6, 0, 0, 0, 0, loc),
		},
		{
			name: "max of several",
			tasks: []Task{
				{Description: "early", Steps: 1, Deadline: "2026-04-05"},
				{Description: "late", Steps: 1, Deadline: "2026-04-10"},
				{Description: "mid", Steps: 1, Deadline: "2026-04-07"},
			},
			want: time.Date(2026, 4, 10, 0, 0, 0, 0, loc),
		},
		{
			name: "ignores empty deadline",
			tasks: []Task{
				{Description: "no due", Steps: 1, Deadline: ""},
				{Description: "due", Steps: 1, Deadline: "2026-04-05"},
			},
			want: time.Date(2026, 4, 5, 0, 0, 0, 0, loc),
		},
		{
			name:    "invalid deadline",
			tasks:   []Task{{Description: "bad", Steps: 1, Deadline: "not-a-date"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := latestDeadlineDay(tt.tasks, loc)
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

func TestWriteViewSchedule(t *testing.T) {
	loc := time.Local

	t.Run("prints days and tasks due", func(t *testing.T) {
		var buf bytes.Buffer
		today := time.Date(2026, 4, 4, 12, 30, 0, 0, loc)
		tasks := []Task{
			{Description: "A", Steps: 1, Deadline: "2026-04-06"},
			{Description: "B", Steps: 1, Deadline: "2026-04-05"},
		}
		if err := writeViewSchedule(&buf, tasks, today); err != nil {
			t.Fatal(err)
		}
		want := strings.TrimSpace(`
2026-04-04
2026-04-05  B [1]
2026-04-06  A [1]`)
		if got := strings.TrimSpace(buf.String()); got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("same day two tasks", func(t *testing.T) {
		var buf bytes.Buffer
		today := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)
		tasks := []Task{
			{Description: "One", Steps: 1, Deadline: "2026-04-04"},
			{Description: "Two", Steps: 1, Deadline: "2026-04-04"},
		}
		if err := writeViewSchedule(&buf, tasks, today); err != nil {
			t.Fatal(err)
		}
		want := strings.TrimSpace(`
2026-04-04  One [1]
2026-04-04  Two [1]`)
		if got := strings.TrimSpace(buf.String()); got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("ddmm deadline lands on correct calendar day", func(t *testing.T) {
		var buf bytes.Buffer
		today := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)
		tasks := []Task{{Description: "x", Steps: 1, Deadline: "05/04/2026"}}
		if err := writeViewSchedule(&buf, tasks, today); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "2026-04-05  x [1]") {
			t.Fatalf("output: %q", buf.String())
		}
	})

	t.Run("no deadlines error", func(t *testing.T) {
		if err := writeViewSchedule(&bytes.Buffer{}, []Task{{Description: "x", Steps: 1}}, time.Now()); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("latest before today", func(t *testing.T) {
		today := time.Date(2026, 4, 10, 0, 0, 0, 0, loc)
		tasks := []Task{{Description: "past", Steps: 1, Deadline: "2026-04-04"}}
		if err := writeViewSchedule(&bytes.Buffer{}, tasks, today); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("interpolates steps across span skips intermediate days", func(t *testing.T) {
		var buf bytes.Buffer
		today := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)
		tasks := []Task{{Description: "bananas", Steps: 3, Deadline: "2026-04-06"}}
		if err := writeViewSchedule(&buf, tasks, today); err != nil {
			t.Fatal(err)
		}
		want := strings.TrimSpace(`
2026-04-01  bananas [1]
2026-04-02
2026-04-03  bananas [2]
2026-04-04
2026-04-05
2026-04-06  bananas [3]`)
		if got := strings.TrimSpace(buf.String()); got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("progress skips finished milestones only remaining interpolated", func(t *testing.T) {
		var buf bytes.Buffer
		today := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)
		tasks := []Task{{Description: "bananas", Steps: 3, Progress: 1, Deadline: "2026-04-06"}}
		if err := writeViewSchedule(&buf, tasks, today); err != nil {
			t.Fatal(err)
		}
		want := strings.TrimSpace(`
2026-04-01  bananas [2]
2026-04-02
2026-04-03
2026-04-04
2026-04-05
2026-04-06  bananas [3]`)
		if got := strings.TrimSpace(buf.String()); got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("one step left places single checkpoint on deadline", func(t *testing.T) {
		var buf bytes.Buffer
		today := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)
		tasks := []Task{{Description: "almost", Steps: 10, Progress: 9, Deadline: "2026-04-06"}}
		if err := writeViewSchedule(&buf, tasks, today); err != nil {
			t.Fatal(err)
		}
		want := strings.TrimSpace(`
2026-04-01
2026-04-02
2026-04-03
2026-04-04
2026-04-05
2026-04-06  almost [10]`)
		if got := strings.TrimSpace(buf.String()); got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("finished task adds no schedule lines", func(t *testing.T) {
		var buf bytes.Buffer
		today := time.Date(2026, 4, 4, 0, 0, 0, 0, loc)
		tasks := []Task{
			{Description: "done", Steps: 3, Progress: 3, Deadline: "2026-04-06"},
			{Description: "todo", Steps: 1, Deadline: "2026-04-06"},
		}
		if err := writeViewSchedule(&buf, tasks, today); err != nil {
			t.Fatal(err)
		}
		want := strings.TrimSpace(`
2026-04-04
2026-04-05
2026-04-06  todo [1]`)
		if got := strings.TrimSpace(buf.String()); got != want {
			t.Fatalf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("many steps uses cumulative targets one line per day", func(t *testing.T) {
		var buf bytes.Buffer
		today := time.Date(2026, 4, 1, 0, 0, 0, 0, loc)
		tasks := []Task{{Description: "book", Steps: 300, Deadline: "2026-04-30"}}
		if err := writeViewSchedule(&buf, tasks, today); err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		var bookLines []string
		seenDay := make(map[string]bool)
		for _, line := range lines {
			line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
			if !strings.Contains(line, "book [") {
				continue
			}
			bookLines = append(bookLines, line)
			day := strings.Fields(line)[0]
			if seenDay[day] {
				t.Fatalf("duplicate day for book: %s", line)
			}
			seenDay[day] = true
		}
		if len(bookLines) != 30 {
			t.Fatalf("want 30 book lines, got %d", len(bookLines))
		}
		if want := "2026-04-01  book [10]"; bookLines[0] != want {
			t.Fatalf("first line got %q want %q", bookLines[0], want)
		}
		if want := "2026-04-30  book [300]"; bookLines[len(bookLines)-1] != want {
			t.Fatalf("last line got %q want %q", bookLines[len(bookLines)-1], want)
		}
	})

	t.Run("more steps than days cumulative without same-day duplicates", func(t *testing.T) {
		byDay := make(map[string][]string)
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
		appendTaskSchedule(byDay, "book", 10, 0, 5, start)
		if len(byDay) != 5 {
			t.Fatalf("want 5 distinct days, got %d: %#v", len(byDay), byDay)
		}
		for i := 0; i < 5; i++ {
			d := start.AddDate(0, 0, i).Format(time.DateOnly)
			entries := byDay[d]
			if len(entries) != 1 {
				t.Fatalf("day %s: want 1 entry, got %v", d, entries)
			}
			want := fmt.Sprintf("book [%d]", (i+1)*2)
			if entries[0] != want {
				t.Fatalf("day %s: got %q want %q", d, entries[0], want)
			}
		}
	})

	t.Run("cumulative spreads only remaining from current progress", func(t *testing.T) {
		byDay := make(map[string][]string)
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
		appendTaskSchedule(byDay, "book", 10, 4, 5, start)
		wants := []int{6, 7, 8, 9, 10}
		for i := 0; i < 5; i++ {
			d := start.AddDate(0, 0, i).Format(time.DateOnly)
			entries := byDay[d]
			if len(entries) != 1 {
				t.Fatalf("day %s: want 1 entry, got %v", d, entries)
			}
			want := fmt.Sprintf("book [%d]", wants[i])
			if entries[0] != want {
				t.Fatalf("day %s: got %q want %q", d, entries[0], want)
			}
		}
	})
}

func TestMilestoneDayOffset(t *testing.T) {
	tests := []struct {
		name   string
		k      int
		steps  int
		dDays  int
		wantOff int
	}{
		{"3 steps 6 days k1", 1, 3, 6, 0},
		{"3 steps 6 days k2", 2, 3, 6, 2},
		{"3 steps 6 days k3", 3, 3, 6, 5},
		{"1 step any span last day", 1, 1, 6, 5},
		{"2 steps 5 days", 2, 2, 5, 4},
		{"single day", 1, 3, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := milestoneDayOffset(tt.k, tt.steps, tt.dDays); got != tt.wantOff {
				t.Fatalf("got %d want %d", got, tt.wantOff)
			}
		})
	}
}

func TestCmdView(t *testing.T) {
	t.Run("invalid args", func(t *testing.T) {
		if err := cmdView([]string{"-nope"}); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAddTask(t *testing.T) {
	t.Run("saves task with required deadline", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		in := strings.NewReader("Buy milk\n2\n2026-06-01\n")
		if err := addTask(in, path); err != nil {
			t.Fatal(err)
		}
		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("len got %d want 1", len(got))
		}
		want := Task{Description: "Buy milk", Steps: 2, Deadline: "2026-06-01"}
		if got[0] != want {
			t.Fatalf("got %+v want %+v", got[0], want)
		}
	})

	t.Run("rejects empty deadline then accepts", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		in := strings.NewReader("Do thing\n1\n\n04/04/2026\n")
		if err := addTask(in, path); err != nil {
			t.Fatal(err)
		}
		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("len got %d want 1", len(got))
		}
		if got[0].Deadline != "2026-04-04" {
			t.Fatalf("deadline got %q", got[0].Deadline)
		}
	})

	t.Run("rejects invalid deadline then accepts", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "tasks.json")
		in := strings.NewReader("x\n1\nnot-a-date\n2026-12-31\n")
		if err := addTask(in, path); err != nil {
			t.Fatal(err)
		}
		got, err := loadTasks(path)
		if err != nil {
			t.Fatal(err)
		}
		if got[0].Deadline != "2026-12-31" {
			t.Fatalf("deadline got %q", got[0].Deadline)
		}
	})
}
