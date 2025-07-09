package wormhole

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

type CloudflareDNSManager struct {
	api     *cloudflare.API
	baseDNS string
	ipV4    string
	zoneID  string
}

func NewCloudflareAPI(apiToken, zoneID, baseDNS, ipv4 string) (DNSAPI, error) {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare API client: %w", err)
	}

	return &CloudflareDNSManager{
		api:     api,
		zoneID:  zoneID,
		baseDNS: baseDNS,
		ipV4:    ipv4,
	}, nil
}

func (d *CloudflareDNSManager) CreateDNSRecord(ctx context.Context, expires time.Duration, record *Record) (*DNSRecord, error) {
	r := cloudflare.CreateDNSRecordParams{
		Type:    string(record.Type),
		Name:    string(record.Name),
		Content: string(record.Content),
		TTL:     int(record.TTL),
	}

	resp, err := d.api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(d.zoneID), r)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS record: %w", err)
	}

	dnsRecord := &DNSRecord{
		Meta: record,
		ID:   resp.ID,
	}

	return dnsRecord, nil
}

func (d *CloudflareDNSManager) DeleteDNSRecord(ctx context.Context, recordID string) error {
	err := d.api.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(d.zoneID), recordID)
	if err != nil {
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}

	return nil
}

func (d *CloudflareDNSManager) IPV4() string    { return d.ipV4 }
func (d *CloudflareDNSManager) BaseDNS() string { return d.baseDNS }
