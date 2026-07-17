package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/DedSecInside/gotor/internal/crawl"
)

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

// Job is the public snapshot returned by the job-control API.
type Job struct {
	ID          string        `json:"id"`
	Status      JobStatus     `json:"status"`
	Request     crawl.Request `json:"request"`
	CreatedAt   time.Time     `json:"createdAt"`
	StartedAt   *time.Time    `json:"startedAt,omitempty"`
	CompletedAt *time.Time    `json:"completedAt,omitempty"`
	Report      *crawl.Report `json:"report,omitempty"`
	Error       string        `json:"error,omitempty"`
}

type managedJob struct {
	job    Job
	cancel context.CancelFunc
}

// JobManager owns crawl lifecycles independently of HTTP request lifetimes.
type JobManager struct {
	mu   sync.RWMutex
	jobs map[string]*managedJob
}

func NewJobManager() *JobManager {
	return &JobManager{jobs: make(map[string]*managedJob)}
}

func newJobID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func (m *JobManager) Start(request crawl.Request) (Job, error) {
	if err := request.Validate(); err != nil {
		// Execute applies proxy defaults, so mirror those defaults for validation
		// at the HTTP boundary.
		if request.SOCKS5Host == "" {
			request.SOCKS5Host = "127.0.0.1"
		}
		if request.SOCKS5Port == 0 {
			request.SOCKS5Port = 9050
		}
		if err = request.Validate(); err != nil {
			return Job{}, err
		}
	}

	id, err := newJobID()
	if err != nil {
		return Job{}, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	state := &managedJob{
		job: Job{
			ID:        id,
			Status:    JobQueued,
			Request:   request,
			CreatedAt: time.Now().UTC(),
		},
		cancel: cancel,
	}

	m.mu.Lock()
	m.jobs[id] = state
	m.mu.Unlock()

	snapshot := state.job
	go m.run(ctx, id)
	return snapshot, nil
}

func (m *JobManager) run(ctx context.Context, id string) {
	startedAt := time.Now().UTC()
	m.mu.Lock()
	state, ok := m.jobs[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	if state.job.Status == JobCancelled {
		m.mu.Unlock()
		return
	}
	state.job.Status = JobRunning
	state.job.StartedAt = &startedAt
	request := state.job.Request
	m.mu.Unlock()

	report, err := crawl.Execute(ctx, request)
	completedAt := time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok = m.jobs[id]
	if !ok {
		return
	}
	state.job.CompletedAt = &completedAt
	state.job.Report = &report
	switch {
	case state.job.Status == JobCancelled || errors.Is(err, context.Canceled):
		state.job.Status = JobCancelled
	case err != nil:
		state.job.Status = JobFailed
		state.job.Error = err.Error()
	default:
		state.job.Status = JobCompleted
	}
}

func (m *JobManager) Get(id string) (Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, ok := m.jobs[id]
	if !ok {
		return Job{}, false
	}
	return state.job, true
}

func (m *JobManager) Cancel(id string) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.jobs[id]
	if !ok {
		return Job{}, false
	}
	switch state.job.Status {
	case JobQueued, JobRunning:
		state.job.Status = JobCancelled
		state.cancel()
	}
	return state.job, true
}

func (m *JobManager) CancelAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, state := range m.jobs {
		switch state.job.Status {
		case JobQueued, JobRunning:
			state.job.Status = JobCancelled
			state.cancel()
		}
	}
}
