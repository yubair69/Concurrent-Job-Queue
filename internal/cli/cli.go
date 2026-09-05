package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func Run(args []string) int {
	if len(args) < 2 {
		printUsage()
		return 1
	}

	cmd := args[1]
	switch cmd {
	case "submit":
		return handleSubmit(args[2:])
	case "status":
		return handleStatus(args[2:])
	case "cancel":
		return handleCancel(args[2:])
	case "workers":
		return handleWorkers(args[2:])
	case "help", "-h", "--help":
		printUsage()
		return 0
	default:
		fmt.Printf("Unknown command: %s\n\n", cmd)
		printUsage()
		return 1
	}
}

func printUsage() {
	fmt.Println("gotask CLI - concurrent background job queue client")
	fmt.Println("\nUsage:")
	fmt.Println("  gotask <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  submit   Submit a new job")
	fmt.Println("  status   Get job status")
	fmt.Println("  cancel   Cancel a job")
	fmt.Println("  workers  List worker and queue metrics")
	fmt.Println("\nRun 'gotask <command> -h' for command-specific flags.")
}

func handleSubmit(args []string) int {
	fs := flag.NewFlagSet("submit", flag.ContinueOnError)
	jobType := fs.String("type", "", "Job type (required)")
	payload := fs.String("payload", "{}", "JSON payload string or @filename")
	priority := fs.Int("priority", 0, "Job priority (higher runs first)")
	maxAttempts := fs.Int("retries", 3, "Maximum retry attempts")
	server := fs.String("server", getServerURL(), "Server URL")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *jobType == "" {
		fmt.Println("Error: --type flag is required")
		fs.Usage()
		return 1
	}

	var jsonPayload json.RawMessage
	payloadStr := *payload
	if strings.HasPrefix(payloadStr, "@") {
		filename := strings.TrimPrefix(payloadStr, "@")
		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error reading payload file '%s': %v\n", filename, err)
			return 1
		}
		jsonPayload = json.RawMessage(data)
	} else {
		if payloadStr == "" {
			payloadStr = "{}"
		}
		if !json.Valid([]byte(payloadStr)) {
			fmt.Printf("Error: invalid JSON payload: %s\n", payloadStr)
			return 1
		}
		jsonPayload = json.RawMessage(payloadStr)
	}

	reqBody := map[string]any{
		"type":         *jobType,
		"payload":      jsonPayload,
		"priority":     *priority,
		"max_attempts": *maxAttempts,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error marshalling request: %v\n", err)
		return 1
	}

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(http.MethodPost, *server+"/jobs", bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return 1
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error connecting to server at %s: %v\n", *server, err)
		return 1
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Server error (status %d): %s\n", resp.StatusCode, string(respBytes))
		return 1
	}

	var result map[string]any
	if err := json.Unmarshal(respBytes, &result); err != nil {
		fmt.Printf("Successfully submitted, but failed to parse response: %s\n", string(respBytes))
		return 0
	}

	fmt.Println("Job successfully submitted:")
	fmt.Printf("  ID:       %v\n", result["id"])
	fmt.Printf("  Type:     %v\n", result["type"])
	fmt.Printf("  Status:   %v\n", result["status"])
	fmt.Printf("  Priority: %v\n", result["priority"])
	return 0
}

func handleStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	server := fs.String("server", getServerURL(), "Server URL")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	tailArgs := fs.Args()
	if len(tailArgs) < 1 {
		fmt.Println("Error: job ID is required")
		fmt.Println("Usage: gotask status <job-id>")
		return 1
	}
	jobID := tailArgs[0]

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/jobs/%s", *server, jobID))
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("Job not found: %s\n", jobID)
		return 1
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server error (status %d): %s\n", resp.StatusCode, string(respBytes))
		return 1
	}

	var job map[string]any
	if err := json.Unmarshal(respBytes, &job); err != nil {
		fmt.Printf("Failed to parse job response: %s\n", string(respBytes))
		return 1
	}

	fmt.Printf("Job Details:\n")
	fmt.Printf("  ID:           %v\n", job["id"])
	fmt.Printf("  Type:         %v\n", job["type"])
	fmt.Printf("  Status:       %v\n", job["status"])
	fmt.Printf("  Priority:     %v\n", job["priority"])
	fmt.Printf("  Attempts:     %v (Max: %v)\n", job["attempts"], job["max_attempts"])
	fmt.Printf("  Created At:   %v\n", job["created_at"])
	fmt.Printf("  Updated At:   %v\n", job["updated_at"])
	fmt.Printf("  Run At:       %v\n", job["run_at"])
	if start, ok := job["started_at"]; ok && start != nil {
		fmt.Printf("  Started At:   %v\n", start)
	}
	if comp, ok := job["completed_at"]; ok && comp != nil {
		fmt.Printf("  Completed At: %v\n", comp)
	}
	if errStr, ok := job["last_error"]; ok && errStr != "" {
		fmt.Printf("  Last Error:   %v\n", errStr)
	}
	return 0
}

func handleCancel(args []string) int {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	server := fs.String("server", getServerURL(), "Server URL")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	tailArgs := fs.Args()
	if len(tailArgs) < 1 {
		fmt.Println("Error: job ID is required")
		fmt.Println("Usage: gotask cancel <job-id>")
		return 1
	}
	jobID := tailArgs[0]

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/jobs/%s", *server, jobID), nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return 1
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("Job not found: %s\n", jobID)
		return 1
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server error (status %d): %s\n", resp.StatusCode, string(respBytes))
		return 1
	}

	var result map[string]any
	if err := json.Unmarshal(respBytes, &result); err != nil {
		fmt.Printf("Job %s cancelled successfully\n", jobID)
		return 0
	}

	fmt.Printf("Job %s cancelled successfully (Status: %v)\n", result["id"], result["status"])
	return 0
}

func handleWorkers(args []string) int {
	fs := flag.NewFlagSet("workers", flag.ContinueOnError)
	server := fs.String("server", getServerURL(), "Server URL")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/metrics", *server))
	if err != nil {
		fmt.Printf("Error connecting to server: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server error (status %d): %s\n", resp.StatusCode, string(respBytes))
		return 1
	}

	var metrics map[string]any
	if err := json.Unmarshal(respBytes, &metrics); err != nil {
		fmt.Printf("Failed to parse metrics: %s\n", string(respBytes))
		return 1
	}

	fmt.Println("GoTask System Metrics & Workers:")
	fmt.Printf("  Active Workers:   %v\n", metrics["active_workers"])
	fmt.Printf("  Queued Jobs:      %v\n", metrics["queued"])
	fmt.Printf("  Running Jobs:     %v\n", metrics["running"])
	fmt.Printf("  Succeeded Jobs:   %v\n", metrics["succeeded"])
	fmt.Printf("  Failed Jobs:      %v\n", metrics["failed"])
	fmt.Printf("  Cancelled Jobs:   %v\n", metrics["cancelled"])
	fmt.Printf("  Total Processed:  %v\n", metrics["total_processed"])
	return 0
}

func getServerURL() string {
	if envServer := os.Getenv("GOTASK_SERVER"); envServer != "" {
		return envServer
	}
	return "http://localhost:8080"
}
