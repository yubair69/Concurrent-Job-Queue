package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("gotask CLI - concurrent background job queue")
		fmt.Println("Usage: gotask <command> [arguments]")
		fmt.Println("Commands:")
		fmt.Println("  submit   Submit a new job")
		fmt.Println("  status   Get job status")
		fmt.Println("  cancel   Cancel a job")
		fmt.Println("  workers  List workers")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "submit", "status", "cancel", "workers":
		fmt.Printf("command '%s' not yet implemented in Phase 1 baseline\n", cmd)
	default:
		fmt.Printf("unknown command: %s\n", cmd)
		os.Exit(1)
	}
}
