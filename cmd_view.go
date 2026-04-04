package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// latestDeadlineDay returns the latest calendar day among tasks with a non-empty deadline, in loc.
func latestDeadlineDay(tasks []Task, loc *time.Location) (time.Time, error) {
	var max time.Time
	var found bool
	for _, t := range tasks {
		if strings.TrimSpace(t.Deadline) == "" {
			continue
		}
		d, err := parseDeadline(t.Deadline)
		if err != nil {
			return time.Time{}, fmt.Errorf("task %q: invalid deadline %q: %w", t.Description, t.Deadline, err)
		}
		day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		if !found || day.After(max) {
			max = day
			found = true
		}
	}
	if !found {
		return time.Time{}, errors.New("no tasks with a deadline")
	}
	return max, nil
}

// writeViewSchedule prints each calendar day from today through lastDeadline (inclusive): a line "YYYY-MM-DD"
// when no tasks are due, or one line per due task "YYYY-MM-DD  <description>".
func writeViewSchedule(w io.Writer, tasks []Task, today time.Time) error {
	loc := today.Location()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	last, err := latestDeadlineDay(tasks, loc)
	if err != nil {
		return err
	}
	if last.Before(start) {
		return fmt.Errorf("latest deadline %s is before today %s", last.Format(time.DateOnly), start.Format(time.DateOnly))
	}
	byDay := make(map[string][]string)
	for _, t := range tasks {
		if strings.TrimSpace(t.Deadline) == "" {
			continue
		}
		d, err := parseDeadline(t.Deadline)
		if err != nil {
			return fmt.Errorf("task %q: invalid deadline %q: %w", t.Description, t.Deadline, err)
		}
		day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		key := day.Format(time.DateOnly)
		byDay[key] = append(byDay[key], t.Description)
	}
	for d := start; !d.After(last); d = d.AddDate(0, 0, 1) {
		key := d.Format(time.DateOnly)
		descs := byDay[key]
		if len(descs) == 0 {
			fmt.Fprintln(w, key)
			continue
		}
		for _, desc := range descs {
			fmt.Fprintf(w, "%s  %s\n", key, desc)
		}
	}
	return nil
}

func cmdView(args []string) error {
	tasksPath, err := parseTasksFilePathArgs(args)
	if err != nil {
		return err
	}
	tasks, err := loadTasks(tasksPath)
	if err != nil {
		return err
	}
	return writeViewSchedule(os.Stdout, tasks, time.Now())
}
