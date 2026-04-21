package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func addTask(r io.Reader, tasksPath string) error {
	br := bufio.NewReader(r)

	var desc string
	var err error
	for desc == "" {
		fmt.Print("Description: ")
		desc, err = readLine(br)
		if err != nil {
			return fmt.Errorf("read description: %w", err)
		}
		if desc == "" {
			fmt.Fprintln(os.Stderr, "Description cannot be empty.")
		}
	}

	var steps int
	for {
		fmt.Print("#steps: ")
		line, err := readLine(br)
		if err != nil {
			return fmt.Errorf("read #steps: %w", err)
		}
		n, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil || n <= 0 {
			fmt.Fprintln(os.Stderr, "#steps must be a positive integer.")
			continue
		}
		steps = n
		break
	}

	var deadlineStr string
	for {
		fmt.Print("Deadline (YYYY-MM-DD or dd/mm/yyyy): ")
		line, err := readLine(br)
		if err != nil {
			return fmt.Errorf("read deadline: %w", err)
		}
		ds, err := deadlineFromInput(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}
		if ds == "" {
			fmt.Fprintln(os.Stderr, "Deadline is required.")
			continue
		}
		deadlineStr = ds
		if err := warnIfDeadlineInPast(os.Stderr, deadlineStr, time.Now()); err != nil {
			return fmt.Errorf("deadline: %w", err)
		}
		break
	}

	var alertDelta int
	for {
		fmt.Print("Alert when delta above (steps, 0=off) [0]: ")
		line, err := readLine(br)
		if err != nil {
			return fmt.Errorf("read alert when delta above: %w", err)
		}
		s := strings.TrimSpace(line)
		if s == "" {
			break
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			fmt.Fprintln(os.Stderr, "Must be a non-negative integer (or empty for 0).")
			continue
		}
		alertDelta = n
		break
	}

	tasks, err := loadTasks(tasksPath)
	if err != nil {
		return err
	}
	tasks = append(tasks, Task{
		Description:         desc,
		Steps:               steps,
		Deadline:            deadlineStr,
		AlertWhenDeltaAbove: alertDelta,
	})
	if err := saveTasks(tasksPath, tasks); err != nil {
		return fmt.Errorf("write %s: %w", tasksPath, err)
	}
	fmt.Printf("Task added (%d task(s) in %s).\n", len(tasks), tasksPath)
	return nil
}

func cmdAdd(args []string) error {
	tasksPath, err := parseAddArgs(args)
	if err != nil {
		return err
	}
	return addTask(os.Stdin, tasksPath)
}
