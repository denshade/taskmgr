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

// interpolatedExpectedProgressEndOfToday returns the ideal absolute progress by the end of the first
// calendar day of the plan (today), aligned with appendTaskSchedule: cumulative targets when remaining
// work exceeds dDays, otherwise linear interpolation from current progress toward the first remaining
// milestone across the day offset to that milestone (when rem == 1, the milestone is on the last day).
func interpolatedExpectedProgressEndOfToday(steps, progress, dDays int) float64 {
	if dDays < 1 {
		dDays = 1
	}
	if steps < 1 {
		steps = 1
	}
	if progress < 0 {
		progress = 0
	}
	if progress >= steps {
		return float64(steps)
	}
	rem := steps - progress
	if rem > dDays {
		add := (rem + dDays - 1) / dDays // ceil(rem / dDays)
		exp := progress + add
		if exp > steps {
			exp = steps
		}
		return float64(exp)
	}
	if dDays == 1 {
		return float64(steps)
	}
	if rem == 1 {
		// Single milestone on last day: interpolate from (0, progress) to (dDays-1, steps).
		return float64(progress) + float64(steps-progress)/float64(dDays-1)
	}
	minOff := -1
	for k := progress + 1; k <= steps; k++ {
		j := k - progress
		off := milestoneDayOffset(j, rem, dDays)
		if minOff < 0 || off < minOff {
			minOff = off
		}
	}
	if minOff == 0 {
		maxK := progress
		for k := progress + 1; k <= steps; k++ {
			j := k - progress
			if milestoneDayOffset(j, rem, dDays) == 0 && k > maxK {
				maxK = k
			}
		}
		return float64(maxK)
	}
	firstK := steps
	for k := progress + 1; k <= steps; k++ {
		j := k - progress
		if milestoneDayOffset(j, rem, dDays) == minOff {
			firstK = k
			break
		}
	}
	return float64(progress) + float64(firstK-progress)/float64(minOff)
}

// appendTaskSchedule adds view lines for one task into byDay. Keys are YYYY-MM-DD in loc.
// Only work left (steps - progress) is scheduled; labels [n/steps] are absolute step totals.
// If remaining work > dDays, targets are cumulative (ceil) from current progress to steps—e.g.
// 300 steps with 0 progress in 30 days yields [10], [20], … [300].
// Otherwise each remaining milestone is placed on an interpolated day (remaining ≤ dDays).
func appendTaskSchedule(byDay map[string][]string, desc string, steps, progress, dDays int, start time.Time) {
	if dDays < 1 {
		return
	}
	if steps < 1 {
		steps = 1
	}
	if progress < 0 {
		progress = 0
	}
	if progress >= steps {
		return
	}
	rem := steps - progress
	loc := start.Location()
	if rem > dDays {
		prev := progress
		for off := 0; off < dDays; off++ {
			d := off + 1
			var target int
			if d == dDays {
				if steps <= prev {
					continue
				}
				target = steps
			} else {
				target = progress + (d*rem+dDays-1)/dDays // progress + ceil(d*rem/dDays)
				if target <= prev {
					continue
				}
			}
			prev = target
			day := start.AddDate(0, 0, off)
			key := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc).Format(time.DateOnly)
			byDay[key] = append(byDay[key], fmt.Sprintf("%s %s", desc, progressLabel(target, steps)))
		}
		return
	}
	for k := progress + 1; k <= steps; k++ {
		j := k - progress
		off := milestoneDayOffset(j, rem, dDays)
		day := start.AddDate(0, 0, off)
		key := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc).Format(time.DateOnly)
		byDay[key] = append(byDay[key], fmt.Sprintf("%s %s", desc, progressLabel(k, steps)))
	}
}

func writeBehindScheduleWarnings(w io.Writer, tasks []Task, start time.Time, loc *time.Location) {
	for _, t := range tasks {
		if t.AlertWhenDeltaAbove <= 0 {
			continue
		}
		if strings.TrimSpace(t.Deadline) == "" {
			continue
		}
		d, err := parseDeadline(t.Deadline)
		if err != nil {
			continue
		}
		endDay := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		if endDay.Before(start) {
			continue
		}
		steps := t.Steps
		if steps < 1 {
			steps = 1
		}
		progress := t.Progress
		if progress < 0 {
			progress = 0
		}
		if progress >= steps {
			continue
		}
		dDays := int(endDay.Sub(start)/(24*time.Hour)) + 1
		interp := interpolatedExpectedProgressEndOfToday(steps, progress, dDays)
		delta := interp - float64(progress)
		if delta < float64(t.AlertWhenDeltaAbove) {
			continue
		}
		fmt.Fprintf(w, "Warning: task %q is behind the interpolated plan (progress %d, expected ≈ %.2f by end of today, delta %.2f; alert when delta >= %d).\n",
			t.Description, progress, interp, delta, t.AlertWhenDeltaAbove)
	}
}

// writeViewSchedule prints each calendar day from today through lastDeadline (inclusive): a line "YYYY-MM-DD"
// when nothing is scheduled, or one line per scheduled checkpoint "YYYY-MM-DD  <description> [n/steps]".
// When remaining steps (#steps − current progress) exceed the days until the deadline, n is the cumulative
// total due by that day (spread with ceil). Otherwise n is each absolute milestone still to reach, on interpolated days.
func progressLabel(current, total int) string {
	return fmt.Sprintf("[%d/%d]", current, total)
}

func writeViewSchedule(w io.Writer, warnOut io.Writer, tasks []Task, today time.Time) error {
	loc := today.Location()
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	last, err := latestDeadlineDay(tasks, loc)
	if err != nil {
		return err
	}
	if last.Before(start) {
		return fmt.Errorf("latest deadline %s is before today %s", last.Format(time.DateOnly), start.Format(time.DateOnly))
	}
	if warnOut != nil {
		writeBehindScheduleWarnings(warnOut, tasks, start, loc)
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
		appendTaskSchedule(byDay, t.Description, steps, t.Progress, dDays, start)
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
	return writeViewSchedule(os.Stdout, os.Stderr, tasks, time.Now())
}
