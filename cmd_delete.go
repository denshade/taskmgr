package main

import "fmt"

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
