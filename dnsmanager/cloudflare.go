package dnsmanager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go/v3"
	"github.com/cloudflare/cloudflare-go/v3/dns"
	"github.com/cloudflare/cloudflare-go/v3/option"
)

var (
	ErrFailedToCreateNewDNSRecord         = errors.New("failed to create new dns record")
	ErrFailedToDeleteDNSRecord            = errors.New("failed to delete dns record")
	ErrFailedToInitializeCloudflareClient = errors.New("failed to initialize cloudflare client")
)

type CloudflareDNSManager struct {
	api     *cloudflare.Client
	baseDNS string
	ipV4    string
	zoneID  string
}

func NewCloudflareAPI(apiToken, zoneID, baseDNS, ipv4 string) (DNSAPI, error) {
	api := cloudflare.NewClient(
		option.WithAPIKey(apiToken),
	)

	return &CloudflareDNSManager{
		api:     api,
		zoneID:  zoneID,
		baseDNS: baseDNS,
		ipV4:    ipv4,
	}, nil
}

func (d *CloudflareDNSManager) CreateDNSRecord(ctx context.Context, ttl time.Duration, record *Record) (*DNSRecord, error) {
	resp, err := d.api.DNS.Records.New(ctx, dns.RecordNewParams{
		ZoneID: cloudflare.F(d.zoneID),
		Record: dns.RecordParam{
			Name:    cloudflare.F(record.Name),
			TTL:     cloudflare.F(dns.TTL(record.TTL)),
			Proxied: cloudflare.F(true),
			Settings: cloudflare.F(any(map[string]any{
				"type":    record.Type,
				"content": record.Content,
			})),
		},
	})
	if err != nil {
		return nil, ErrFailedToCreateNewDNSRecord
	}

	dnsRecord := &DNSRecord{
		Meta: record,
		ID:   resp.ID,
		TTL:  ttl,
	}

	return dnsRecord, nil
}

func (d *CloudflareDNSManager) DeleteDNSRecord(ctx context.Context, recordID string) error {
	_, err := d.api.DNS.Records.Delete(
		ctx,
		recordID,
		dns.RecordDeleteParams{},
	)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToDeleteDNSRecord, err)
	}

	return nil
}

func (d *CloudflareDNSManager) IPV4() string    { return d.ipV4 }
func (d *CloudflareDNSManager) BaseDNS() string { return d.baseDNS }
