// Package store provides data stores for the TUI.
package store

import (
	"sync"

	"github.com/Dyastin-0/wormhole/core/proto"
	"github.com/Dyastin-0/wormhole/stream"
)

type LogStore struct {
	mu      sync.RWMutex
	logs    map[string]*stream.HTTPEvent
	order   []string
	maxLogs int
}

func New(maxLogs int) *LogStore {
	return &LogStore{
		logs:    make(map[string]*stream.HTTPEvent, maxLogs),
		order:   make([]string, 0, maxLogs),
		maxLogs: maxLogs,
	}
}

func (s *LogStore) AddEvent(event *stream.HTTPEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs[event.ID] = event
	s.order = append(s.order, event.ID)

	if len(s.order) > s.maxLogs {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.logs, oldest)
	}
}

func (s *LogStore) AddDuration(log *proto.HTTPDurationLog) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event, ok := s.logs[log.ID]; ok {
		event.Duration = log.Duration
	}
}

func (s *LogStore) Get(id string) *stream.HTTPEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.logs[id]
}

func (s *LogStore) GetByIndex(index int) *stream.HTTPEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 || index >= len(s.order) {
		return nil
	}
	return s.logs[s.order[index]]
}

func (s *LogStore) GetRange(start, end int) []*stream.HTTPEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if start < 0 {
		start = 0
	}
	if end > len(s.order) {
		end = len(s.order)
	}
	if start >= end {
		return nil
	}

	ids := s.order[start:end]
	result := make([]*stream.HTTPEvent, 0, len(ids))
	for _, id := range ids {
		if event, ok := s.logs[id]; ok {
			result = append(result, event)
		}
	}
	return result
}

func (s *LogStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.order)
}

func (s *LogStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs = make(map[string]*stream.HTTPEvent, s.maxLogs)
	s.order = s.order[:0]
}
