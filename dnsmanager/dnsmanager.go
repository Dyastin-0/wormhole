// Package dnsmanager
package dnsmanager

import (
	"context"
	"time"
)

type DNSManager interface {
	BaseDomain() string
	IPV4() string
	CreateDNSRecord(ctx context.Context, ttl time.Duration, record *Record) (*DNSRecord, error)
	DeleteDNSRecord(ctx context.Context, id string) error
}
