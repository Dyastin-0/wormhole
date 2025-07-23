package dnsmanager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

var (
	ErrFailedToCreateNewDNSRecord         = errors.New("failed to create new dns record")
	ErrFailedToDeleteDNSRecord            = errors.New("failed to delete dns record")
	ErrFailedToInitializeCloudflareClient = errors.New("failed to initialize cloudflare client")
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
		return nil, fmt.Errorf("%v: %w", ErrFailedToInitializeCloudflareClient, err)
	}

	return &CloudflareDNSManager{
		api:     api,
		zoneID:  zoneID,
		baseDNS: baseDNS,
		ipV4:    ipv4,
	}, nil
}

func (d *CloudflareDNSManager) CreateDNSRecord(ctx context.Context, ttl time.Duration, record *Record) (*DNSRecord, error) {
	r := cloudflare.CreateDNSRecordParams{
		Type:    string(record.Type),
		Name:    string(record.Name),
		Content: string(record.Content),
		TTL:     int(record.TTL),
	}

	resp, err := d.api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(d.zoneID), r)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrFailedToCreateNewDNSRecord, err)
	}

	dnsRecord := &DNSRecord{
		Meta: record,
		ID:   resp.ID,
		TTL:  ttl,
	}

	return dnsRecord, nil
}

func (d *CloudflareDNSManager) DeleteDNSRecord(ctx context.Context, recordID string) error {
	err := d.api.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(d.zoneID), recordID)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrFailedToDeleteDNSRecord, err)
	}

	return nil
}

func (d *CloudflareDNSManager) IPV4() string    { return d.ipV4 }
func (d *CloudflareDNSManager) BaseDNS() string { return d.baseDNS }
