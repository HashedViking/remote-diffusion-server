package fileserver

import (
	"sync"
	"time"
)

type DownloadJobStatus string

const (
	PENDING     DownloadJobStatus = "PENDING"
	DOWNLOADING DownloadJobStatus = "DOWNLOADING"
	CANCELED    DownloadJobStatus = "CANCELED"
	COMPLETED   DownloadJobStatus = "COMPLETED"
	FAILED      DownloadJobStatus = "FAILED"
)

type DownloadJob struct {
	ID        int               `json:"id"`
	Status    DownloadJobStatus `json:"status"`
	UserKey   string            `json:"userKey"`
	FileName  string            `json:"fileName"`
	FilePath  string            `json:"filePath"`
	Url       string            `json:"url"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

func NewDownloadJob() DownloadJob {
	return DownloadJob{}
}

type DownloadJobQueue struct {
	mu   sync.Mutex
	jobs map[string][]*DownloadJob
}

func NewDownloadJobQueue() DownloadJobQueue {
	return DownloadJobQueue{
		jobs: make(map[string][]*DownloadJob),
	}
}

func (q *DownloadJobQueue) AddJob(userKey string, job DownloadJob) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.jobs[userKey] = append(q.jobs[userKey], &job)
}

func (q *DownloadJobQueue) GetJobs(userKey string) []*DownloadJob {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.jobs[userKey]
}

func (q *DownloadJobQueue) JobAlreadyExists(userKey string, filePath string, url string) *DownloadJob {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, job := range q.jobs[userKey] {
		if job.FilePath == filePath &&
			job.Url == url &&
			job.Status != COMPLETED &&
			job.Status != FAILED {
			return job
		}
	}
	return nil
}

func (q *DownloadJobQueue) GetJobByID(userKey string, id int) *DownloadJob {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, job := range q.jobs[userKey] {
		if job.ID == id {
			return job
		}
	}
	return nil
}
