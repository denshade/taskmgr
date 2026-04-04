package main

import (
	"fmt"
	"io"
	"os"
)

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
