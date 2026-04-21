package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

const defaultTasksFile = "tasks.json"

type Task struct {
	Description         string `json:"description"`
	Progress            int    `json:"progress,omitempty"`
	Steps               int    `json:"steps"`
	Deadline            string `json:"deadline,omitempty"`
	AlertWhenDeltaAbove int    `json:"alert_when_delta_above,omitempty"`
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
