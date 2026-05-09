package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// progressNeededTodayLabel returns the value to display after "Progress needed today:".
// It returns "" when the task has no (parseable) deadline so the line is omitted.
//
// The number reported is today's share of the remaining work assuming a flat linear pace
// from today through the deadline (inclusive): (steps − progress) / dDays. This is the
// continuous rate, not the discrete milestone schedule that `view` prints, so values are
// fractional in the common case (e.g. 28 steps over 84 days → 0.33). When the deadline
// is in the past with work remaining, all remaining work is annotated as overdue. When
// the task is already done, the label notes that explicitly.
func progressNeededTodayLabel(t Task, today time.Time) string {
	if strings.TrimSpace(t.Deadline) == "" {
		return ""
	}
	d, err := parseDeadline(t.Deadline)
	if err != nil {
		return ""
	}
	loc := today.Location()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	endDay := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
	steps := t.Steps
	if steps < 1 {
		steps = 1
	}
	progress := t.Progress
	if progress < 0 {
		progress = 0
	}
	if progress >= steps {
		return "0 (done)"
	}
	if endDay.Before(start) {
		return fmt.Sprintf("%d (overdue)", steps-progress)
	}
	dDays := int(endDay.Sub(start)/(24*time.Hour)) + 1
	if dDays < 1 {
		dDays = 1
	}
	delta := float64(steps-progress) / float64(dDays)
	return fmt.Sprintf("%.2f", delta)
}

func writeTaskList(w io.Writer, tasks []Task, today time.Time) {
	if len(tasks) == 0 {
		fmt.Fprintln(w, "No tasks.")
		return
	}
	for i, t := range tasks {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "Index: %d\n", i+1)
		fmt.Fprintf(w, "Description: %s\n", t.Description)
		fmt.Fprintf(w, "Current progress: %d\n", t.Progress)
		fmt.Fprintf(w, "#steps: %d\n", t.Steps)
		deadline := t.Deadline
		if deadline == "" {
			deadline = "(none)"
		}
		fmt.Fprintf(w, "Deadline: %s\n", deadline)
		alert := t.AlertWhenDeltaAbove
		if alert == 0 {
			fmt.Fprintln(w, "Alert when delta above: (off)")
		} else {
			fmt.Fprintf(w, "Alert when delta above: %d\n", alert)
		}
		if label := progressNeededTodayLabel(t, today); label != "" {
			fmt.Fprintf(w, "Progress needed today: %s\n", label)
		}
	}
}

func writeTasksFromPath(w io.Writer, tasksPath string, today time.Time) error {
	tasks, err := loadTasks(tasksPath)
	if err != nil {
		return err
	}
	writeTaskList(w, tasks, today)
	return nil
}

func cmdList(args []string) error {
	tasksPath, err := parseTasksFilePathArgs(args)
	if err != nil {
		return err
	}
	return writeTasksFromPath(os.Stdout, tasksPath, time.Now())
}
