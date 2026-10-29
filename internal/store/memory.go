package store

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("task not found")

type Task struct {
	ID          uuid.UUID
	Title       string
	Description string
	Status      string
	DueDate     *time.Time
	CreatedAt   time.Time
}

type MemoryStore struct {
	mu    sync.RWMutex
	tasks map[uuid.UUID]Task
	order []uuid.UUID
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tasks: make(map[uuid.UUID]Task)}
}

func (s *MemoryStore) Create(t Task) Task {
	t.ID = uuid.New()
	t.CreatedAt = time.Now().UTC().Truncate(time.Second)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t
	s.order = append(s.order, t.ID)
	return t
}

func (s *MemoryStore) Get(id uuid.UUID) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

// List возвращает страницу задач и общее количество задач с таким статусом.
func (s *MemoryStore) List(status string, limit, offset int) ([]Task, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []Task
	for _, id := range s.order {
		t := s.tasks[id]
		if status == "" || t.Status == status {
			filtered = append(filtered, t)
		}
	}

	total := len(filtered)
	if offset >= total {
		return []Task{}, total
	}
	end := min(offset+limit, total)
	return filtered[offset:end], total
}
