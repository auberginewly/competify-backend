package memory

import (
	"sync"

	"github.com/competify-ai/competify-backend/internal/schema"
)

// ReportStore is a thread-safe in-memory store for final reports.
type ReportStore struct {
	mu   sync.RWMutex
	data map[string]schema.FinalReport
}

// NewReportStore creates a new ReportStore.
func NewReportStore() *ReportStore {
	return &ReportStore{data: make(map[string]schema.FinalReport)}
}

// Save stores a final report.
func (s *ReportStore) Save(r schema.FinalReport) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[r.TaskID] = r
}

// Get retrieves a report by task ID.
func (s *ReportStore) Get(taskID string) (schema.FinalReport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.data[taskID]
	return r, ok
}
