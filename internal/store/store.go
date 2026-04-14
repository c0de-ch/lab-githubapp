package store

type ListRunsOpts struct {
	Owner  string
	Repo   string
	Status string
	Limit  int
	Offset int
}

type Store interface {
	// Migrations
	Migrate() error

	// Installations
	UpsertInstallation(inst *Installation) error
	ListInstallations() ([]Installation, error)
	DeleteInstallation(installationID int64) error
	GetInstallationForRepo(owner, repo string) (*Installation, error)

	// Repositories
	UpsertRepository(repo *Repository) error
	ListRepositories(installationID int64) ([]Repository, error)
	ListAllRepositories() ([]Repository, error)
	DeleteRepository(repoID int64) error
	GetRepository(owner, repo string) (*Repository, error)

	// Workflow Runs
	UpsertWorkflowRun(run *WorkflowRun) error
	GetWorkflowRun(runID int64) (*WorkflowRun, error)
	ListWorkflowRuns(opts ListRunsOpts) ([]WorkflowRun, error)
	ListRecentRuns(owner, repo string, limit int) ([]WorkflowRun, error)

	// Workflow Jobs
	UpsertWorkflowJob(job *WorkflowJob) error
	ListWorkflowJobs(runID int64) ([]WorkflowJob, error)

	// Close
	Close() error
}
