package github

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

type LogEntry struct {
	JobName  string
	StepName string
	Content  string
}

func (c *Client) GetRunLogs(ctx context.Context, owner, repo string, runID int64) ([]LogEntry, error) {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return nil, err
	}
	client := c.auth.ClientForInstallation(instID)

	url, _, err := client.Actions.GetWorkflowRunLogs(ctx, owner, repo, runID, 4)
	if err != nil {
		return nil, fmt.Errorf("getting log URL: %w", err)
	}

	resp, err := http.Get(url.String())
	if err != nil {
		return nil, fmt.Errorf("downloading logs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading log response: %w", err)
	}

	return extractLogEntries(body)
}

func (c *Client) GetJobLogs(ctx context.Context, owner, repo string, jobID int64) (string, error) {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return "", err
	}
	client := c.auth.ClientForInstallation(instID)

	url, _, err := client.Actions.GetWorkflowJobLogs(ctx, owner, repo, jobID, 4)
	if err != nil {
		return "", fmt.Errorf("getting job log URL: %w", err)
	}

	resp, err := http.Get(url.String())
	if err != nil {
		return "", fmt.Errorf("downloading job logs: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading job log response: %w", err)
	}

	return string(data), nil
}

func extractLogEntries(zipData []byte) ([]LogEntry, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	var entries []LogEntry

	// Sort files by name for consistent ordering
	files := make([]*zip.File, len(reader.File))
	copy(files, reader.File)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		// GitHub log zip structure: "job-name/step-number_step-name.txt"
		jobName, stepName := parseLogFileName(f.Name)

		entries = append(entries, LogEntry{
			JobName:  jobName,
			StepName: stepName,
			Content:  string(content),
		})
	}

	return entries, nil
}

func parseLogFileName(name string) (string, string) {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) < 2 {
		return name, ""
	}

	jobName := parts[0]
	stepFile := parts[1]

	// Remove .txt extension
	stepFile = strings.TrimSuffix(stepFile, ".txt")

	// Remove leading step number (e.g., "1_Set up job")
	if idx := strings.Index(stepFile, "_"); idx >= 0 {
		stepFile = stepFile[idx+1:]
	}

	return jobName, stepFile
}
