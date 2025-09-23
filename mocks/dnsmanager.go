// Package mocks
package mocks

import (
	"context"
	"time"

	"github.com/Dyastin-0/wormhole/dnsmanager"
	"github.com/stretchr/testify/mock"
)

type DNSManager struct {
	mock.Mock
}

func (m *DNSManager) CreateDNSRecord(ctx context.Context, ttl time.Duration, record *dnsmanager.Record) (*dnsmanager.DNSRecord, error) {
	args := m.Called(ctx, ttl, record)
	return args.Get(0).(*dnsmanager.DNSRecord), args.Error(1)
}

func (m *DNSManager) DeleteDNSRecord(ctx context.Context, recordID string) error {
	args := m.Called(ctx, recordID)
	return args.Error(0)
}

func (m *DNSManager) BaseDomain() string {
	args := m.Called()
	return args.String(0)
}

func (m *DNSManager) IPV4() string {
	args := m.Called()
	return args.String()
}
