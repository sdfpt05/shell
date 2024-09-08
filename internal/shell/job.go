package shell

import (
	"os/exec"
)

type Job struct {
	Command    *exec.Cmd
	Status     string
	ID         int
	Background bool
}

func (s *Shell) CreateJob(cmd *exec.Cmd, background bool) *Job {
	job := &Job{
		Command:    cmd,
		Status:     "Running",
		ID:         s.nextJobID,
		Background: background,
	}
	s.jobs[s.nextJobID] = job
	s.nextJobID++
	return job
}

func (s *Shell) ListJobs() []*Job {
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}
