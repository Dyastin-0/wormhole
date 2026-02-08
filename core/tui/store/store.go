// Package store provides data stores for the TUI.
package store

import (
	"sync"

	"github.com/Dyastin-0/wormhole/core/tui/messages"
)

type LogStore struct {
	mu      sync.RWMutex
	logs    []*messages.HTTPLogMsg
	maxLogs int
}

func New(maxLogs int) *LogStore {
	return &LogStore{
		logs:    make([]*messages.HTTPLogMsg, 0, maxLogs),
		maxLogs: maxLogs,
	}
}

func (s *LogStore) Add(log *messages.HTTPLogMsg) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs = append(s.logs, log)
	if len(s.logs) > s.maxLogs {
		s.logs = s.logs[len(s.logs)-s.maxLogs:]
	}
}

func (s *LogStore) Get(index int) *messages.HTTPLogMsg {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index >= 0 && index < len(s.logs) {
		return s.logs[index]
	}
	return nil
}

func (s *LogStore) GetRange(start, end int) []*messages.HTTPLogMsg {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if start < 0 {
		start = 0
	}
	if end > len(s.logs) {
		end = len(s.logs)
	}
	if start >= end {
		return nil
	}

	// Return a copy of the slice header to prevent
	// external modification issues
	return s.logs[start:end]
}

func (s *LogStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.logs)
}

func (s *LogStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = s.logs[:0]
}
