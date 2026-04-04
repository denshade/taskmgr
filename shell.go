package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// runCLICommand runs a single subcommand (first token after program name). handled is false if cmd is unknown.
func runCLICommand(cmd string, args []string) (handled bool, err error) {
	switch cmd {
	case "help", "-h", "--help":
		printHelp()
		return true, nil
	case "add":
		return true, cmdAdd(args)
	case "delete":
		return true, cmdDelete(args)
	case "list":
		return true, cmdList(args)
	case "edit":
		return true, cmdEdit(args)
	case "view":
		return true, cmdView(args)
	default:
		return false, nil
	}
}

// splitShellLine splits a line into fields; spaces inside double quotes are preserved.
func splitShellLine(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var fields []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' && !inQuote:
			inQuote = true
		case c == '"' && inQuote:
			inQuote = false
		case (c == ' ' || c == '\t') && !inQuote:
			if b.Len() > 0 {
				fields = append(fields, b.String())
				b.Reset()
			}
		default:
			b.WriteByte(c)
		}
	}
	if b.Len() > 0 || (len(fields) > 0 && inQuote) {
		fields = append(fields, b.String())
	}
	return fields
}

func runShell() {
	runShellWith(os.Stdin, os.Stdout, os.Stderr)
}

func runShellWith(in io.Reader, out io.Writer, errOut io.Writer) {
	br := bufio.NewReader(in)
	fmt.Fprintln(out, "deadline — interactive shell; same commands as `deadline <command>`. Type exit or press Ctrl+D to quit.")
	for {
		fmt.Fprint(out, "deadline> ")
		line, err := readLine(br)
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(out)
				return
			}
			fmt.Fprintf(errOut, "read input: %v\n", err)
			return
		}
		args := splitShellLine(line)
		if len(args) == 0 {
			continue
		}
		switch args[0] {
		case "exit", "quit":
			return
		}
		cmd := args[0]
		rest := args[1:]
		handled, err := runCLICommand(cmd, rest)
		if !handled {
			fmt.Fprintf(errOut, "unknown command: %q (type help for usage)\n", cmd)
			continue
		}
		if err != nil {
			fmt.Fprintf(errOut, "%s: %v\n", cmd, err)
		}
	}
}
