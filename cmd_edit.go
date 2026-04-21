package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

func validateProgress(progress, steps int) error {
	if progress < 0 {
		return errors.New("progress cannot be negative")
	}
	if progress > steps {
		return fmt.Errorf("progress (%d) cannot exceed #steps (%d)", progress, steps)
	}
	return nil
}

func editTask(r io.Reader, tasksPath string, idx1 int) error {
	br := bufio.NewReader(r)
	tasks, err := loadTasks(tasksPath)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks in %s", tasksPath)
	}
	i := idx1 - 1
	if i >= len(tasks) {
		return fmt.Errorf("no task at index %d (%d task(s) in %s)", idx1, len(tasks), tasksPath)
	}
	t := tasks[i]

	fmt.Printf("Description[%q]: ", t.Description)
	line, err := readLine(br)
	if err != nil {
		return fmt.Errorf("read description: %w", err)
	}
	if line != "" {
		t.Description = line
	}

	for {
		fmt.Printf("Current progress[%d]: ", t.Progress)
		line, err := readLine(br)
		if err != nil {
			return fmt.Errorf("read progress: %w", err)
		}
		if line == "" {
			if err := validateProgress(t.Progress, t.Steps); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			break
		}
		p, err := strconv.Atoi(line)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Current progress must be an integer.")
			continue
		}
		if err := validateProgress(p, t.Steps); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}
		t.Progress = p
		break
	}

	for {
		fmt.Printf("#steps[%d]: ", t.Steps)
		line, err := readLine(br)
		if err != nil {
			return fmt.Errorf("read #steps: %w", err)
		}
		if line == "" {
			if err := validateProgress(t.Progress, t.Steps); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			break
		}
		n, err := strconv.Atoi(line)
		if err != nil || n <= 0 {
			fmt.Fprintln(os.Stderr, "#steps must be a positive integer.")
			continue
		}
		t.Steps = n
		if err := validateProgress(t.Progress, t.Steps); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}
		break
	}

	for {
		if t.Deadline == "" {
			fmt.Print(`Deadline["(none)"]: `)
		} else {
			fmt.Printf("Deadline[%q]: ", t.Deadline)
		}
		line, err := readLine(br)
		if err != nil {
			return fmt.Errorf("read deadline: %w", err)
		}
		if line == "" {
			break
		}
		ds, err := deadlineFromInput(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}
		if ds == "" {
			fmt.Fprintln(os.Stderr, "Deadline is required (enter a date, or press Enter to keep the current value).")
			continue
		}
		t.Deadline = ds
		if err := warnIfDeadlineInPast(os.Stderr, t.Deadline, time.Now()); err != nil {
			return fmt.Errorf("deadline: %w", err)
		}
		break
	}

	for {
		fmt.Printf("Alert when delta above (0=off)[%d]: ", t.AlertWhenDeltaAbove)
		line, err := readLine(br)
		if err != nil {
			return fmt.Errorf("read alert when delta above: %w", err)
		}
		if line == "" {
			break
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 0 {
			fmt.Fprintln(os.Stderr, "Must be a non-negative integer (or Enter to keep the current value).")
			continue
		}
		t.AlertWhenDeltaAbove = n
		break
	}

	tasks[i] = t
	if err := saveTasks(tasksPath, tasks); err != nil {
		return fmt.Errorf("write %s: %w", tasksPath, err)
	}
	if err := appendTaskEditHistory(tasksPath, time.Now(), t.Description, t.Progress, t.Steps); err != nil {
		return fmt.Errorf("append history: %w", err)
	}
	fmt.Printf("Task %d edited (%d task(s) in %s).\n", idx1, len(tasks), tasksPath)
	return nil
}

func cmdEdit(args []string) error {
	tasksPath, idx1, err := parseDeleteArgs(args)
	if err != nil {
		return err
	}
	return editTask(os.Stdin, tasksPath, idx1)
}
