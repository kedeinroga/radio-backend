package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// JobScheduler maneja la ejecución de trabajos programados
type JobScheduler struct {
	cron   *cron.Cron
	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	jobs   map[string]*JobInfo
	mu     sync.RWMutex
}

// JobInfo contiene información sobre un job registrado
type JobInfo struct {
	Name         string        `json:"name"`
	Schedule     string        `json:"schedule"`
	EntryID      cron.EntryID  `json:"entry_id"`
	LastRun      time.Time     `json:"last_run"`
	NextRun      time.Time     `json:"next_run"`
	RunCount     int64         `json:"run_count"`
	FailCount    int64         `json:"fail_count"`
	LastError    string        `json:"last_error,omitempty"`
	LastDuration time.Duration `json:"last_duration"`
}

// NewJobScheduler crea un nuevo scheduler de jobs
func NewJobScheduler(logger *slog.Logger) *JobScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	// Cron con segundos habilitados
	c := cron.New(cron.WithSeconds())

	return &JobScheduler{
		cron:   c,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
		jobs:   make(map[string]*JobInfo),
	}
}

// AddJob registra un nuevo job con su schedule
func (s *JobScheduler) AddJob(name, schedule string, job func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, err := s.cron.AddFunc(schedule, func() {
		s.executeJob(name, job)
	})

	if err != nil {
		return fmt.Errorf("failed to add job %s: %w", name, err)
	}

	s.jobs[name] = &JobInfo{
		Name:     name,
		Schedule: schedule,
		EntryID:  entryID,
	}

	s.logger.Info("job registered",
		"name", name,
		"schedule", schedule,
	)

	return nil
}

// executeJob ejecuta un job con manejo de errores y métricas
func (s *JobScheduler) executeJob(name string, job func()) {
	s.wg.Add(1)
	defer s.wg.Done()

	start := time.Now()
	s.logger.Debug("job started", "name", name)

	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("job panicked",
				"name", name,
				"panic", r,
			)
			s.updateJobStats(name, time.Since(start), fmt.Errorf("panic: %v", r))
		}
	}()

	// Ejecutar job
	job()

	duration := time.Since(start)
	s.logger.Info("job completed",
		"name", name,
		"duration", duration,
	)

	s.updateJobStats(name, duration, nil)
}

// updateJobStats actualiza las estadísticas de un job
func (s *JobScheduler) updateJobStats(name string, duration time.Duration, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobInfo, exists := s.jobs[name]
	if !exists {
		return
	}

	jobInfo.LastRun = time.Now()
	jobInfo.LastDuration = duration
	jobInfo.RunCount++

	if err != nil {
		jobInfo.FailCount++
		jobInfo.LastError = err.Error()
	} else {
		jobInfo.LastError = ""
	}

	// Actualizar NextRun desde el cron
	entry := s.cron.Entry(jobInfo.EntryID)
	jobInfo.NextRun = entry.Next
}

// Start inicia el scheduler
func (s *JobScheduler) Start() {
	s.logger.Info("starting job scheduler",
		"job_count", len(s.jobs),
	)
	s.cron.Start()

	// Actualizar NextRun para todos los jobs
	s.mu.Lock()
	for name, jobInfo := range s.jobs {
		entry := s.cron.Entry(jobInfo.EntryID)
		jobInfo.NextRun = entry.Next
		s.logger.Info("job scheduled",
			"name", name,
			"next_run", entry.Next.Format(time.RFC3339),
		)
	}
	s.mu.Unlock()
}

// Stop detiene el scheduler con graceful shutdown
func (s *JobScheduler) Stop() {
	s.logger.Info("stopping job scheduler...")

	// Cancelar contexto
	s.cancel()

	// Detener cron (no acepta nuevos jobs)
	ctx := s.cron.Stop()

	// Esperar a que termine el cron
	<-ctx.Done()

	// Esperar jobs en ejecución
	s.logger.Info("waiting for running jobs to complete...")
	s.wg.Wait()

	s.logger.Info("job scheduler stopped")
}

// GetJobStats retorna estadísticas de todos los jobs
func (s *JobScheduler) GetJobStats() map[string]JobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]JobInfo)
	for name, job := range s.jobs {
		// Actualizar NextRun desde el cron
		entry := s.cron.Entry(job.EntryID)
		job.NextRun = entry.Next

		stats[name] = *job
	}
	return stats
}

// GetJobStat retorna estadísticas de un job específico
func (s *JobScheduler) GetJobStat(name string) (JobInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[name]
	if !exists {
		return JobInfo{}, false
	}

	// Actualizar NextRun desde el cron
	entry := s.cron.Entry(job.EntryID)
	job.NextRun = entry.Next

	return *job, true
}

// RemoveJob elimina un job del scheduler
func (s *JobScheduler) RemoveJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[name]
	if !exists {
		return fmt.Errorf("job %s not found", name)
	}

	s.cron.Remove(job.EntryID)
	delete(s.jobs, name)

	s.logger.Info("job removed", "name", name)
	return nil
}

// Context retorna el contexto del scheduler
func (s *JobScheduler) Context() context.Context {
	return s.ctx
}
