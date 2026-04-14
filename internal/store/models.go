package store

import (
	"encoding/json"
	"time"
)

type Installation struct {
	ID           int64
	AppID        int64
	AccountLogin string
	AccountType  string
	TargetType   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository struct {
	ID             int64
	InstallationID int64
	Owner          string
	Name           string
	FullName       string
	Private        bool
	HTMLURL        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type WorkflowRun struct {
	ID           int64
	RepositoryID int64
	WorkflowID   int64
	WorkflowName string
	RunNumber    int
	HeadBranch   string
	HeadSHA      string
	Event        string
	Status       string
	Conclusion   string
	HTMLURL      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (r *WorkflowRun) ToJSON() string {
	data, _ := json.Marshal(r)
	return string(data)
}

func (r *WorkflowRun) StatusClass() string {
	switch r.Status {
	case "completed":
		switch r.Conclusion {
		case "success":
			return "success"
		case "failure":
			return "failure"
		case "cancelled":
			return "cancelled"
		default:
			return "neutral"
		}
	case "in_progress":
		return "in-progress"
	case "queued":
		return "queued"
	default:
		return "neutral"
	}
}

func (r *WorkflowRun) StatusLabel() string {
	if r.Status == "completed" && r.Conclusion != "" {
		return r.Conclusion
	}
	return r.Status
}

type WorkflowJob struct {
	ID          int64
	RunID       int64
	Name        string
	Status      string
	Conclusion  string
	StartedAt   time.Time
	CompletedAt time.Time
	RunnerName  string
	Steps       string // JSON
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (j *WorkflowJob) ToJSON() string {
	data, _ := json.Marshal(j)
	return string(data)
}

type WebhookEvent struct {
	ID          int64
	EventType   string
	Action      string
	DeliveryID  string
	Payload     string
	ProcessedAt time.Time
}
