// Package dnsmanager
package dnsmanager

import (
	"context"
	"fmt"
	"time"
)

type DNSAPI interface {
	BaseDNS() string
	IPV4() string
	CreateDNSRecord(ctx context.Context, ttl time.Duration, record *Record) (*DNSRecord, error)
	DeleteDNSRecord(ctx context.Context, id string) error
}

type Manager struct {
	API DNSAPI
}

func NewCloudflareManager(apiToken, zone, baseDNS, ipv4 string) (*Manager, error) {
	api, err := NewCloudflareAPI(apiToken, zone, baseDNS, ipv4)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", "failed to initialize cloudflare manager", err)
	}

	manager := &Manager{
		API: api,
	}

	return manager, nil
}

func (m *Manager) WatchExpiration() {}
