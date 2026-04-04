package main

import "fmt"

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
