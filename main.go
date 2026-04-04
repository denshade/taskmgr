package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		runShell()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]
	if handled, err := runCLICommand(cmd, args); handled {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", cmd, err)
			os.Exit(1)
		}
		return
	}
	fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", cmd)
	printHelp()
	os.Exit(1)
}
