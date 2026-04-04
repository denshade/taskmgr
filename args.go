package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

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

// parseDeleteArgs returns the tasks file path and a 1-based task index (delete and edit).
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
