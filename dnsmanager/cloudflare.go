package dnsmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CloudflareDNSAPI struct {
	apiToken string
	baseDNS  string
	ipV4     string
	zoneID   string
}

var (
	ErrFailedToCreateNewDNSRecord         = errors.New("failed to create new dns record")
	ErrFailedToDeleteDNSRecord            = errors.New("failed to delete dns record")
	ErrFailedToInitializeCloudflareClient = errors.New("failed to initialize cloudflare client")
)

func NewCloudflareAPI(apiToken, zoneID, baseDNS, ipv4 string) DNSAPI {
	return &CloudflareDNSAPI{
		apiToken: apiToken,
		zoneID:   zoneID,
		baseDNS:  baseDNS,
		ipV4:     ipv4,
	}
}

type cloudflareDNSCreateRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type cloudflareDNSCreateResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Result struct {
		ID string `json:"id"`
	} `json:"result"`
}

func (c *CloudflareDNSAPI) CreateDNSRecord(ctx context.Context, ttl time.Duration, record *Record) (*DNSRecord, error) {
	body := cloudflareDNSCreateRequest{
		Type:    string(record.Type),
		Name:    string(record.Name),
		Content: string(record.Content),
		TTL:     record.TTL,
		Proxied: record.Proxied,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", c.zoneID),
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, ErrFailedToCreateNewDNSRecord
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var parsed cloudflareDNSCreateResponse
	if err := json.Unmarshal(data, &parsed); err != nil || !parsed.Success {
		return nil, fmt.Errorf("%w: %s", ErrFailedToCreateNewDNSRecord, string(data))
	}

	return &DNSRecord{
		Meta: record,
		ID:   parsed.Result.ID,
		TTL:  ttl,
	}, nil
}

func (c *CloudflareDNSAPI) DeleteDNSRecord(ctx context.Context, recordID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", c.zoneID, recordID),
		nil,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFailedToDeleteDNSRecord, err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: %s", ErrFailedToDeleteDNSRecord, string(data))
	}

	return nil
}

func (c *CloudflareDNSAPI) BaseDNS() string { return c.baseDNS }
func (c *CloudflareDNSAPI) IPV4() string    { return c.ipV4 }
