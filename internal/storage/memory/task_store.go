package memory

import (
	"sync"
	"time"
)

// TaskStatus represents the lifecycle state of a task.
type TaskStatus struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"` // pending / running / done / error
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// TaskStore is a thread-safe in-memory store for task statuses.
type TaskStore struct {
	mu   sync.RWMutex
	data map[string]TaskStatus
}

// NewTaskStore creates a new TaskStore.
func NewTaskStore() *TaskStore {
	return &TaskStore{data: make(map[string]TaskStatus)}
}

// Create initializes a task with "pending" status.
func (s *TaskStore) Create(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	s.data[taskID] = TaskStatus{
		TaskID:    taskID,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Update modifies the status of an existing task.
func (s *TaskStore) Update(taskID, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ts, ok := s.data[taskID]; ok {
		ts.Status = status
		ts.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		s.data[taskID] = ts
	}
}

// Get retrieves the status of a task.
func (s *TaskStore) Get(taskID string) (TaskStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ts, ok := s.data[taskID]
	return ts, ok
}

// List returns all tasks (for debug/admin).
func (s *TaskStore) List() []TaskStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TaskStatus, 0, len(s.data))
	for _, ts := range s.data {
		out = append(out, ts)
	}
	return out
}
