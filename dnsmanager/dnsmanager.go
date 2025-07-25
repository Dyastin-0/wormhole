// Package dnsmanager
package dnsmanager

import (
	"context"
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

func NewCloudflareManager(apiToken, zone, baseDNS, ipv4 string) *Manager {
	api := NewCloudflareAPI(apiToken, zone, baseDNS, ipv4)
	manager := &Manager{
		API: api,
	}

	return manager
}

func (m *Manager) WatchExpiration() {}
