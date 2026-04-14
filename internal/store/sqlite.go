package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) Migrate() error {
	_, err := s.db.Exec(migrationSQL)
	return err
}

const migrationSQL = `
CREATE TABLE IF NOT EXISTS installations (
    id              INTEGER PRIMARY KEY,
    app_id          INTEGER NOT NULL,
    account_login   TEXT NOT NULL,
    account_type    TEXT NOT NULL,
    target_type     TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS repositories (
    id              INTEGER PRIMARY KEY,
    installation_id INTEGER NOT NULL REFERENCES installations(id) ON DELETE CASCADE,
    owner           TEXT NOT NULL,
    name            TEXT NOT NULL,
    full_name       TEXT NOT NULL,
    private         BOOLEAN NOT NULL DEFAULT FALSE,
    html_url        TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_repos_installation ON repositories(installation_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_repos_fullname ON repositories(full_name);

CREATE TABLE IF NOT EXISTS workflow_runs (
    id              INTEGER PRIMARY KEY,
    repository_id   INTEGER NOT NULL,
    workflow_id     INTEGER NOT NULL,
    workflow_name   TEXT NOT NULL,
    run_number      INTEGER NOT NULL,
    head_branch     TEXT NOT NULL,
    head_sha        TEXT NOT NULL,
    event           TEXT NOT NULL,
    status          TEXT NOT NULL,
    conclusion      TEXT NOT NULL DEFAULT '',
    html_url        TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runs_repo ON workflow_runs(repository_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON workflow_runs(status);

CREATE TABLE IF NOT EXISTS workflow_jobs (
    id              INTEGER PRIMARY KEY,
    run_id          INTEGER NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL,
    conclusion      TEXT NOT NULL DEFAULT '',
    started_at      DATETIME,
    completed_at    DATETIME,
    runner_name     TEXT NOT NULL DEFAULT '',
    steps           TEXT NOT NULL DEFAULT '[]',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_jobs_run ON workflow_jobs(run_id);
`

// --- Installations ---

func (s *SQLiteStore) UpsertInstallation(inst *Installation) error {
	_, err := s.db.Exec(`
		INSERT INTO installations (id, app_id, account_login, account_type, target_type, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			account_login = excluded.account_login,
			account_type = excluded.account_type,
			target_type = excluded.target_type,
			updated_at = excluded.updated_at`,
		inst.ID, inst.AppID, inst.AccountLogin, inst.AccountType, inst.TargetType, time.Now(),
	)
	return err
}

func (s *SQLiteStore) ListInstallations() ([]Installation, error) {
	rows, err := s.db.Query(`SELECT id, app_id, account_login, account_type, target_type, created_at, updated_at FROM installations ORDER BY account_login`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []Installation
	for rows.Next() {
		var inst Installation
		if err := rows.Scan(&inst.ID, &inst.AppID, &inst.AccountLogin, &inst.AccountType, &inst.TargetType, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
			return nil, err
		}
		installations = append(installations, inst)
	}
	return installations, rows.Err()
}

func (s *SQLiteStore) DeleteInstallation(installationID int64) error {
	_, err := s.db.Exec(`DELETE FROM installations WHERE id = ?`, installationID)
	return err
}

func (s *SQLiteStore) GetInstallationForRepo(owner, repo string) (*Installation, error) {
	var inst Installation
	err := s.db.QueryRow(`
		SELECT i.id, i.app_id, i.account_login, i.account_type, i.target_type, i.created_at, i.updated_at
		FROM installations i
		JOIN repositories r ON r.installation_id = i.id
		WHERE r.owner = ? AND r.name = ?`,
		owner, repo,
	).Scan(&inst.ID, &inst.AppID, &inst.AccountLogin, &inst.AccountType, &inst.TargetType, &inst.CreatedAt, &inst.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// --- Repositories ---

func (s *SQLiteStore) UpsertRepository(repo *Repository) error {
	_, err := s.db.Exec(`
		INSERT INTO repositories (id, installation_id, owner, name, full_name, private, html_url, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			installation_id = excluded.installation_id,
			owner = excluded.owner,
			name = excluded.name,
			full_name = excluded.full_name,
			private = excluded.private,
			html_url = excluded.html_url,
			updated_at = excluded.updated_at`,
		repo.ID, repo.InstallationID, repo.Owner, repo.Name, repo.FullName, repo.Private, repo.HTMLURL, time.Now(),
	)
	return err
}

func (s *SQLiteStore) ListRepositories(installationID int64) ([]Repository, error) {
	rows, err := s.db.Query(`
		SELECT id, installation_id, owner, name, full_name, private, html_url, created_at, updated_at
		FROM repositories WHERE installation_id = ? ORDER BY full_name`,
		installationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var r Repository
		if err := rows.Scan(&r.ID, &r.InstallationID, &r.Owner, &r.Name, &r.FullName, &r.Private, &r.HTMLURL, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *SQLiteStore) ListAllRepositories() ([]Repository, error) {
	rows, err := s.db.Query(`
		SELECT id, installation_id, owner, name, full_name, private, html_url, created_at, updated_at
		FROM repositories ORDER BY full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var r Repository
		if err := rows.Scan(&r.ID, &r.InstallationID, &r.Owner, &r.Name, &r.FullName, &r.Private, &r.HTMLURL, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *SQLiteStore) DeleteRepository(repoID int64) error {
	_, err := s.db.Exec(`DELETE FROM repositories WHERE id = ?`, repoID)
	return err
}

func (s *SQLiteStore) GetRepository(owner, repo string) (*Repository, error) {
	var r Repository
	err := s.db.QueryRow(`
		SELECT id, installation_id, owner, name, full_name, private, html_url, created_at, updated_at
		FROM repositories WHERE owner = ? AND name = ?`,
		owner, repo,
	).Scan(&r.ID, &r.InstallationID, &r.Owner, &r.Name, &r.FullName, &r.Private, &r.HTMLURL, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Workflow Runs ---

func (s *SQLiteStore) UpsertWorkflowRun(run *WorkflowRun) error {
	_, err := s.db.Exec(`
		INSERT INTO workflow_runs (id, repository_id, workflow_id, workflow_name, run_number, head_branch, head_sha, event, status, conclusion, html_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			conclusion = excluded.conclusion,
			updated_at = excluded.updated_at`,
		run.ID, run.RepositoryID, run.WorkflowID, run.WorkflowName, run.RunNumber,
		run.HeadBranch, run.HeadSHA, run.Event, run.Status, run.Conclusion,
		run.HTMLURL, run.CreatedAt, run.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) GetWorkflowRun(runID int64) (*WorkflowRun, error) {
	var run WorkflowRun
	err := s.db.QueryRow(`
		SELECT id, repository_id, workflow_id, workflow_name, run_number, head_branch, head_sha, event, status, conclusion, html_url, created_at, updated_at
		FROM workflow_runs WHERE id = ?`,
		runID,
	).Scan(&run.ID, &run.RepositoryID, &run.WorkflowID, &run.WorkflowName, &run.RunNumber,
		&run.HeadBranch, &run.HeadSHA, &run.Event, &run.Status, &run.Conclusion,
		&run.HTMLURL, &run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *SQLiteStore) ListWorkflowRuns(opts ListRunsOpts) ([]WorkflowRun, error) {
	query := `
		SELECT wr.id, wr.repository_id, wr.workflow_id, wr.workflow_name, wr.run_number,
			wr.head_branch, wr.head_sha, wr.event, wr.status, wr.conclusion, wr.html_url,
			wr.created_at, wr.updated_at
		FROM workflow_runs wr
		JOIN repositories r ON r.id = wr.repository_id
		WHERE r.owner = ? AND r.name = ?`

	args := []any{opts.Owner, opts.Repo}

	if opts.Status != "" {
		query += ` AND wr.status = ?`
		args = append(args, opts.Status)
	}

	query += ` ORDER BY wr.updated_at DESC`

	if opts.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, opts.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []WorkflowRun
	for rows.Next() {
		var run WorkflowRun
		if err := rows.Scan(&run.ID, &run.RepositoryID, &run.WorkflowID, &run.WorkflowName, &run.RunNumber,
			&run.HeadBranch, &run.HeadSHA, &run.Event, &run.Status, &run.Conclusion,
			&run.HTMLURL, &run.CreatedAt, &run.UpdatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *SQLiteStore) ListRecentRuns(owner, repo string, limit int) ([]WorkflowRun, error) {
	return s.ListWorkflowRuns(ListRunsOpts{
		Owner: owner,
		Repo:  repo,
		Limit: limit,
	})
}

// --- Workflow Jobs ---

func (s *SQLiteStore) UpsertWorkflowJob(job *WorkflowJob) error {
	_, err := s.db.Exec(`
		INSERT INTO workflow_jobs (id, run_id, name, status, conclusion, started_at, completed_at, runner_name, steps, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			conclusion = excluded.conclusion,
			completed_at = excluded.completed_at,
			runner_name = excluded.runner_name,
			steps = excluded.steps,
			updated_at = excluded.updated_at`,
		job.ID, job.RunID, job.Name, job.Status, job.Conclusion,
		job.StartedAt, job.CompletedAt, job.RunnerName, job.Steps, time.Now(),
	)
	return err
}

func (s *SQLiteStore) ListWorkflowJobs(runID int64) ([]WorkflowJob, error) {
	rows, err := s.db.Query(`
		SELECT id, run_id, name, status, conclusion, started_at, completed_at, runner_name, steps, created_at, updated_at
		FROM workflow_jobs WHERE run_id = ? ORDER BY id`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []WorkflowJob
	for rows.Next() {
		var job WorkflowJob
		if err := rows.Scan(&job.ID, &job.RunID, &job.Name, &job.Status, &job.Conclusion,
			&job.StartedAt, &job.CompletedAt, &job.RunnerName, &job.Steps, &job.CreatedAt, &job.UpdatedAt); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}
