package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

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
