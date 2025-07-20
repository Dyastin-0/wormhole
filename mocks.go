package wormhole

import (
	"context"
	"net"
	"time"

	"github.com/Dyastin-0/wormhole/dnsmanager"
)

type mockSession struct {
	conn net.Conn
	err  error
}

func (m *mockSession) Open() (net.Conn, error) {
	return m.conn, m.err
}

type mockDNSAPI struct {
	baseDNS string
	ipv4    string
}

func newMockDNSAPI(baseDNS, ipv4 string) *mockDNSAPI {
	return &mockDNSAPI{
		baseDNS: baseDNS,
		ipv4:    ipv4,
	}
}

func (m *mockDNSAPI) CreateDNSRecord(context context.Context, expires time.Duration, record *dnsmanager.Record) (*dnsmanager.DNSRecord, error) {
	return &dnsmanager.DNSRecord{
		Meta: &dnsmanager.Record{
			Content: "localhost",
		},
	}, nil
}

func (m *mockDNSAPI) DeleteDNSRecord(context context.Context, id string) error {
	return nil
}

func (m *mockDNSAPI) IPV4() string    { return m.ipv4 }
func (m *mockDNSAPI) BaseDNS() string { return m.baseDNS }
