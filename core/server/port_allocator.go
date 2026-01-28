package server

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"

	"github.com/rs/zerolog/log"
)

// PortAllocator manages TCP port allocation.
type PortAllocator struct {
	mu           sync.Mutex
	minPort      int
	maxPort      int
	maxRandRetry int
	allocated    map[int]bool
}

// NewPortAllocator creates a new port allocator for the given range.
func NewPortAllocator(minPort, maxPort, maxRandRetry int) *PortAllocator {
	return &PortAllocator{
		minPort:      minPort,
		maxPort:      maxPort,
		maxRandRetry: maxRandRetry,
		allocated:    make(map[int]bool, maxPort-minPort+1),
	}
}

// AllocateListener allocates a random available port and creates a listener.
func (pa *PortAllocator) AllocateListener() (int, net.Listener, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	for range pa.maxRandRetry {
		port := pa.minPort + rand.Intn(pa.maxPort-pa.minPort+1)
		if !pa.allocated[port] {
			ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err == nil {
				pa.allocated[port] = true
				return port, ln, nil
			}
		}
	}

	available := make([]int, 0, pa.maxPort-pa.minPort+1-len(pa.allocated))
	for port := pa.minPort; port <= pa.maxPort; port++ {
		if !pa.allocated[port] {
			available = append(available, port)
		}
	}

	if len(available) == 0 {
		return 0, nil, errors.New("no available ports in range")
	}

	rand.Shuffle(len(available), func(i, j int) {
		available[i], available[j] = available[j], available[i]
	})

	for _, port := range available {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			// Port is in available list, but is actually in use (maybe by another running service).
			// We don't mark this as allocated since we won't have a way to release it practically.
			log.Debug().Err(err).Int("port", port).Msg("port marked as available but is in use")
			continue
		}

		pa.allocated[port] = true
		return port, ln, nil
	}

	return 0, nil, errors.New("failed to allocate any available port")
}

// ReleasePort marks a port as available for reuse.
func (pa *PortAllocator) ReleasePort(port int) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	delete(pa.allocated, port)
}

// GetAllocatedCount returns the number of currently allocated ports.
func (pa *PortAllocator) GetAllocatedCount() int {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	return len(pa.allocated)
}
