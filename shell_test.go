package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSplitShellLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace", "  \t  ", nil},
		{"single", "list", []string{"list"}},
		{"two", "delete 2", []string{"delete", "2"}},
		{"quoted path", `add -f "my tasks.json"`, []string{"add", "-f", "my tasks.json"}},
		{"tabs", "list\t-f\tfoo.json", []string{"list", "-f", "foo.json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitShellLine(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("idx %d: got %q want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRunCLICommand_unknown(t *testing.T) {
	handled, err := runCLICommand("nope", nil)
	if handled {
		t.Fatal("expected not handled")
	}
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}
}

func TestRunShellWith(t *testing.T) {
	t.Run("help and exit", func(t *testing.T) {
		in := strings.NewReader("help\nexit\n")
		var out, errOut bytes.Buffer
		runShellWith(in, &out, &errOut)
		if !strings.Contains(out.String(), "deadline —") {
			t.Fatalf("stdout: %q", out.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr: %q", errOut.String())
		}
	})

	t.Run("unknown then exit", func(t *testing.T) {
		in := strings.NewReader("nope\nexit\n")
		var out, errOut bytes.Buffer
		runShellWith(in, &out, &errOut)
		if !strings.Contains(errOut.String(), "unknown command") {
			t.Fatalf("stderr: %q", errOut.String())
		}
	})

	t.Run("quit", func(t *testing.T) {
		in := strings.NewReader("quit\n")
		var out, errOut bytes.Buffer
		runShellWith(in, &out, &errOut)
		if errOut.Len() != 0 {
			t.Fatalf("stderr: %q", errOut.String())
		}
	})

	t.Run("EOF on empty", func(t *testing.T) {
		in := strings.NewReader("")
		var out, errOut bytes.Buffer
		runShellWith(in, &out, &errOut)
		if errOut.Len() != 0 {
			t.Fatalf("stderr: %q", errOut.String())
		}
	})
}
