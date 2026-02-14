package convex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles communication with Convex evaluation platform
type Client struct {
	baseURL   string
	secretKey string
	client    *http.Client
}

// NewClient creates a Convex API client
func NewClient(baseURL, secretKey string) *Client {
	return &Client{
		baseURL:   baseURL,
		secretKey: secretKey,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Task represents an evaluation task with credentials
type Task struct {
	ID          string                 `json:"task_id"`
	RunID       string                 `json:"runId,omitempty"`
	Text        string                 `json:"confirmed_task"`
	Website     string                 `json:"website,omitempty"`
	LoginCookie string                 `json:"loginCookie,omitempty"`
	Credentials map[string]any `json:"credentials,omitempty"`
}

// FetchTestCase fetches tasks from Convex
func (c *Client) FetchTestCase(ctx context.Context, testCaseName string) ([]Task, error) {
	url := fmt.Sprintf("%s/api/getTestCase", c.baseURL)

	payload := map[string]string{"name": testCaseName}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	// Retry logic
	var resp *http.Response
	var err error
	for attempt := range 5 {
		resp, err = c.client.Do(req)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(1<<attempt) * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch test case failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tasks []Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// FetchTask fetches a single task by ID from a test case
func (c *Client) FetchTask(ctx context.Context, testCaseName, taskID string) (*Task, error) {
	tasks, err := c.FetchTestCase(ctx, testCaseName)
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		if task.ID == taskID {
			return &task, nil
		}
	}

	return nil, fmt.Errorf("task %s not found in test case %s", taskID, testCaseName)
}
