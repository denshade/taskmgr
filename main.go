package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultTasksFile = "tasks.json"

type Task struct {
	Description string `json:"description"`
	Progress    int    `json:"progress,omitempty"`
	Steps       int    `json:"steps"`
	Deadline    string `json:"deadline,omitempty"`
}

func printHelp() {
	fmt.Println(`taskmgr — simple task manager

Usage:
  taskmgr <command>

Commands:
  help    Show this message
  add     Add a task (prompts; data in tasks.json, or -f/-file for another JSON path)
  delete  Remove a task by 1-based index (tasks.json, or -f/-file for another JSON path)
  list    List all tasks (tasks.json, or -f/-file for another JSON path)
  update  Edit a task by 1-based index (prompts; Enter keeps each field; -f/-file for JSON path)
  view    Print each day from today through the latest deadline (-f/-file for JSON path)`)
}

func parseDeadline(input string) (time.Time, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return time.Time{}, errors.New("deadline is empty")
	}
	loc := time.Local
	layouts := []string{
		time.DateOnly,
		"02/01/2006",
		"2/01/2006",
		"02/1/2006",
		"2/1/2006",
	}
	var lastErr error
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, s, loc)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, fmt.Errorf("expected ISO date (YYYY-MM-DD) or dd/mm/yyyy: %w", lastErr)
}

// deadlineFromInput returns normalized YYYY-MM-DD, or "" if input is empty (no deadline).
func deadlineFromInput(line string) (string, error) {
	s := strings.TrimSpace(line)
	if s == "" {
		return "", nil
	}
	d, err := parseDeadline(s)
	if err != nil {
		return "", err
	}
	return d.Format(time.DateOnly), nil
}

func loadTasks(path string) ([]Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return tasks, nil
}

func saveTasks(path string, tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// parseTasksFilePathArgs parses -f/-file for commands that only need a tasks JSON path.
func parseTasksFilePathArgs(args []string) (tasksPath string, err error) {
	fs := flag.NewFlagSet("file", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := defaultTasksFile
	fs.StringVar(&path, "f", defaultTasksFile, "path to tasks JSON file")
	fs.StringVar(&path, "file", defaultTasksFile, "path to tasks JSON file")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() != 0 {
		return "", fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("tasks file path is empty")
	}
	return filepath.Clean(path), nil
}

func parseAddArgs(args []string) (tasksPath string, err error) {
	return parseTasksFilePathArgs(args)
}

// parseDeleteArgs returns the tasks file path and a 1-based task index.
func parseDeleteArgs(args []string) (tasksPath string, index1 int, err error) {
	path := defaultTasksFile
	var positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-f" || a == "-file":
			if i+1 >= len(args) {
				return "", 0, fmt.Errorf("flag %s requires a value", a)
			}
			i++
			path = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "-f="):
			path = strings.TrimSpace(strings.TrimPrefix(a, "-f="))
		case strings.HasPrefix(a, "-file="):
			path = strings.TrimSpace(strings.TrimPrefix(a, "-file="))
		case strings.HasPrefix(a, "-"):
			return "", 0, fmt.Errorf("unknown flag %q", a)
		default:
			positionals = append(positionals, a)
		}
	}
	if strings.TrimSpace(path) == "" {
		return "", 0, errors.New("tasks file path is empty")
	}
	path = filepath.Clean(path)
	if len(positionals) != 1 {
		if len(positionals) == 0 {
			return "", 0, errors.New("task index is required (1-based)")
		}
		return "", 0, fmt.Errorf("expected exactly one task index, got %d arguments: %s", len(positionals), strings.Join(positionals, " "))
	}
	n, err := strconv.Atoi(strings.TrimSpace(positionals[0]))
	if err != nil || n < 1 {
		return "", 0, errors.New("task index must be a positive integer")
	}
	return path, n, nil
}

func validateProgress(progress, steps int) error {
	if progress < 0 {
		return errors.New("progress cannot be negative")
	}
	if progress > steps {
		return fmt.Errorf("progress (%d) cannot exceed #steps (%d)", progress, steps)
	}
	return nil
}

func cmdDelete(args []string) error {
	tasksPath, idx1, err := parseDeleteArgs(args)
	if err != nil {
		return err
	}

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

	tasks = append(tasks[:i], tasks[i+1:]...)
	if err := saveTasks(tasksPath, tasks); err != nil {
		return fmt.Errorf("write %s: %w", tasksPath, err)
	}
	fmt.Printf("Task deleted (%d task(s) in %s).\n", len(tasks), tasksPath)
	return nil
}

func writeTaskList(w io.Writer, tasks []Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(w, "No tasks.")
		return
	}
	for i, t := range tasks {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "Description: %s\n", t.Description)
		fmt.Fprintf(w, "Current progress: %d\n", t.Progress)
		fmt.Fprintf(w, "#steps: %d\n", t.Steps)
		deadline := t.Deadline
		if deadline == "" {
			deadline = "(none)"
		}
		fmt.Fprintf(w, "Deadline: %s\n", deadline)
	}
}

func writeTasksFromPath(w io.Writer, tasksPath string) error {
	tasks, err := loadTasks(tasksPath)
	if err != nil {
		return err
	}
	writeTaskList(w, tasks)
	return nil
}

func cmdList(args []string) error {
	tasksPath, err := parseTasksFilePathArgs(args)
	if err != nil {
		return err
	}
	return writeTasksFromPath(os.Stdout, tasksPath)
}

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
		break
	}

	tasks, err := loadTasks(tasksPath)
	if err != nil {
		return err
	}
	tasks = append(tasks, Task{
		Description: desc,
		Steps:       steps,
		Deadline:    deadlineStr,
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

func updateTask(r io.Reader, tasksPath string, idx1 int) error {
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
		fmt.Printf("#steps[%d]: ", t.Steps)
		line, err := readLine(br)
		if err != nil {
			return fmt.Errorf("read #steps: %w", err)
		}
		if line == "" {
			break
		}
		n, err := strconv.Atoi(line)
		if err != nil || n <= 0 {
			fmt.Fprintln(os.Stderr, "#steps must be a positive integer.")
			continue
		}
		t.Steps = n
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
		break
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

	tasks[i] = t
	if err := saveTasks(tasksPath, tasks); err != nil {
		return fmt.Errorf("write %s: %w", tasksPath, err)
	}
	fmt.Printf("Task %d updated (%d task(s) in %s).\n", idx1, len(tasks), tasksPath)
	return nil
}

func cmdUpdate(args []string) error {
	tasksPath, idx1, err := parseDeleteArgs(args)
	if err != nil {
		return err
	}
	return updateTask(os.Stdin, tasksPath, idx1)
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "help", "-h", "--help":
		printHelp()
	case "add":
		if err := cmdAdd(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "add: %v\n", err)
			os.Exit(1)
		}
	case "delete":
		if err := cmdDelete(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "delete: %v\n", err)
			os.Exit(1)
		}
	case "list":
		if err := cmdList(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "list: %v\n", err)
			os.Exit(1)
		}
	case "update":
		if err := cmdUpdate(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "update: %v\n", err)
			os.Exit(1)
		}
	case "view":
		if err := cmdView(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "view: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}
