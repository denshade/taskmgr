package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWarnIfDeadlineInPast(t *testing.T) {
	loc := time.Local
	today := time.Date(2026, 4, 4, 15, 0, 0, 0, loc)

	tests := []struct {
		name      string
		dateOnly  string
		wantWarn  bool
		wantSubstr string
	}{
		{
			name:     "yesterday warns",
			dateOnly: "2026-04-03",
			wantWarn: true,
			wantSubstr: "Warning: deadline 2026-04-03 is in the past",
		},
		{
			name:     "last year warns",
			dateOnly: "2025-04-09",
			wantWarn: true,
			wantSubstr: "today is 2026-04-04",
		},
		{
			name:    "today no warn",
			dateOnly: "2026-04-04",
			wantWarn: false,
		},
		{
			name:    "tomorrow no warn",
			dateOnly: "2026-04-05",
			wantWarn: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := warnIfDeadlineInPast(&buf, tt.dateOnly, today); err != nil {
				t.Fatal(err)
			}
			out := buf.String()
			if tt.wantWarn {
				if out == "" {
					t.Fatal("expected warning on stderr")
				}
				if !strings.Contains(out, tt.wantSubstr) {
					t.Fatalf("output %q should contain %q", out, tt.wantSubstr)
				}
			} else if out != "" {
				t.Fatalf("unexpected output: %q", out)
			}
		})
	}

	t.Run("invalid date returns error", func(t *testing.T) {
		var buf bytes.Buffer
		err := warnIfDeadlineInPast(&buf, "not-a-date", today)
		if err == nil {
			t.Fatal("expected error")
		}
		if buf.Len() != 0 {
			t.Fatalf("unexpected output: %q", buf.String())
		}
	})
}
