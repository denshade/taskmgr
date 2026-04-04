package main

import "fmt"

func helpText() string {
	return `deadline — tasks with steps and due dates

Usage:
  deadline              Start an interactive shell (same commands below; exit or Ctrl+D to quit)
  deadline <command>    Run one command and exit

Commands:
  help    Show this message
  add     Add a task (prompts; data in tasks.json, or -f/-file for another JSON path)
  delete  Remove a task by 1-based index (tasks.json, or -f/-file for another JSON path)
  list    List all tasks (tasks.json, or -f/-file for another JSON path)
  edit    Edit a task by 1-based index (prompts; Enter keeps each field; -f/-file for JSON path)
  view    Through the latest deadline: schedules only work left from current progress; per-step days when remaining ≤ days left, else cumulative targets spread with ceil (-f/-file for JSON path)`
}

func printHelp() {
	fmt.Println(helpText())
}
