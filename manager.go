package wormhole

import (
	"context"
	"fmt"
	"time"
)

type DNSAPI interface {
	CreateDNSRecord(context context.Context, expires time.Duration, record Record) (*DNSRecord, error)
	DeleteDNSRecord(context context.Context, id string) error
}

type Manager struct {
	API DNSAPI
}

func NewCloudflareManager(apiToken, zone string) (*Manager, error) {
	api, err := NewCloudflareAPI(apiToken, zone)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", "failed to initialize cloudflare manager", err)
	}

	manager := &Manager{
		API: api,
	}

	return manager, nil
}

func (m *Manager) WatchExpiration() {}
