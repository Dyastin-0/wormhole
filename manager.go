package wormhole

import (
	"context"
	"time"

	cloudflare_option "github.com/cloudflare/cloudflare-go/v4/option"
)

type DNSAPI interface {
	CreateDNSRecord(context context.Context, expires time.Duration, record Record) (*DNSRecord, error)
	DeleteDNSRecord(context context.Context, tid string) error
}

type Manager struct {
	API DNSAPI
}

func NewCloudflareManager(zone string) *Manager {
	return &Manager{
		API: NewCloudflareAPI(zone),
	}
}

func NewCloudflareManagerWithOpts(zone string, opts ...cloudflare_option.RequestOption) *Manager {
	return &Manager{
		API: NewCloudflareAPIWithOpts(zone, opts...),
	}
}

func (m *Manager) WatchExpiration() {}
