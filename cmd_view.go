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

// milestoneDayOffset returns the 0-based day offset from rangeStart for milestone k (1-based) out of steps
// milestones spread evenly through an inclusive span of dDays calendar days (dDays >= 1).
func milestoneDayOffset(k, steps, dDays int) int {
	if dDays < 1 {
		return 0
	}
	if steps < 1 {
		steps = 1
	}
	if steps == 1 {
		return dDays - 1
	}
	return (k - 1) * (dDays - 1) / (steps - 1)
}

// appendTaskSchedule adds view lines for one task into byDay. Keys are YYYY-MM-DD in loc.
// If steps > dDays, targets are cumulative (ceil) through the span—e.g. 300 steps in 30 days
// yields [10], [20], … [300] on distinct emissions, at most one line per calendar day.
// Otherwise each unit step gets its own interpolated day (steps ≤ dDays).
func appendTaskSchedule(byDay map[string][]string, desc string, steps, dDays int, start time.Time) {
	if dDays < 1 {
		return
	}
	if steps < 1 {
		steps = 1
	}
	loc := start.Location()
	if steps > dDays {
		prev := 0
		for off := 0; off < dDays; off++ {
			d := off + 1
			var target int
			if d == dDays {
				if steps <= prev {
					continue
				}
				target = steps
			} else {
				target = (d*steps + dDays - 1) / dDays // ceil(d*steps/dDays)
				if target <= prev {
					continue
				}
			}
			prev = target
			day := start.AddDate(0, 0, off)
			key := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc).Format(time.DateOnly)
			byDay[key] = append(byDay[key], fmt.Sprintf("%s [%d]", desc, target))
		}
		return
	}
	for k := 1; k <= steps; k++ {
		off := milestoneDayOffset(k, steps, dDays)
		day := start.AddDate(0, 0, off)
		key := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc).Format(time.DateOnly)
		byDay[key] = append(byDay[key], fmt.Sprintf("%s [%d]", desc, k))
	}
}

// writeViewSchedule prints each calendar day from today through lastDeadline (inclusive): a line "YYYY-MM-DD"
// when nothing is scheduled, or one line per scheduled checkpoint "YYYY-MM-DD  <description> [n]".
// When #steps is greater than the number of days until the deadline, n is the cumulative amount due by
// that day (spread with ceil). Otherwise n is the 1..steps milestone index on interpolated days.
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
		endDay := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		if endDay.Before(start) {
			continue
		}
		steps := t.Steps
		if steps < 1 {
			steps = 1
		}
		dDays := int(endDay.Sub(start)/(24*time.Hour)) + 1
		appendTaskSchedule(byDay, t.Description, steps, dDays, start)
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
