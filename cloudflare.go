package wormhole

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

type DNSManager struct {
	api    *cloudflare.API
	zoneID string
}

func NewCloudflareAPI(apiToken, zoneID string) (DNSAPI, error) {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare API client: %w", err)
	}

	return &DNSManager{
		api:    api,
		zoneID: zoneID,
	}, nil
}

func (d *DNSManager) CreateDNSRecord(ctx context.Context, expires time.Duration, record Record) (*DNSRecord, error) {
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

func (d *DNSManager) DeleteDNSRecord(ctx context.Context, recordID string) error {
	err := d.api.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(d.zoneID), recordID)
	if err != nil {
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}

	return nil
}
