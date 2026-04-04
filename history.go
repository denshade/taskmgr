package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func historyCSVPath(tasksPath string) string {
	return filepath.Join(filepath.Dir(tasksPath), "history.csv")
}

// appendTaskEditHistory appends one row to history.csv next to tasksPath.
// Columns: timestamp, task (description), progress, steps. Writes a header row when the file is new or empty.
func appendTaskEditHistory(tasksPath string, ts time.Time, task string, progress, steps int) error {
	path := historyCSVPath(tasksPath)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	needHeader := info.Size() == 0

	w := csv.NewWriter(f)
	if needHeader {
		if err := w.Write([]string{"timestamp", "task", "progress", "steps"}); err != nil {
			return err
		}
	}
	row := []string{
		ts.Format(time.RFC3339),
		task,
		strconv.Itoa(progress),
		strconv.Itoa(steps),
	}
	if err := w.Write(row); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}
