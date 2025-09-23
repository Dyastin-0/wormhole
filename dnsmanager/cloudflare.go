package dnsmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Cloudflare struct {
	token      string
	baseDomain string
	ipV4       string
	zoneID     string
}

var (
	ErrFailedToCreateNewDNSRecord         = errors.New("failed to create new dns record")
	ErrRecordAlreadyExists                = errors.New("record already exists")
	ErrFailedToDeleteDNSRecord            = errors.New("failed to delete dns record")
	ErrFailedToInitializeCloudflareClient = errors.New("failed to initialize cloudflare client")
)

type OptFunc func(c *Cloudflare)

func WithBaseDomain(domain string) OptFunc {
	return func(c *Cloudflare) {
		c.baseDomain = domain
	}
}

func WithToken(token string) OptFunc {
	return func(c *Cloudflare) {
		c.token = token
	}
}

func WithIPv4(ipV4 string) OptFunc {
	return func(c *Cloudflare) {
		c.ipV4 = ipV4
	}
}

func WithZoneID(zoneID string) OptFunc {
	return func(c *Cloudflare) {
		c.zoneID = zoneID
	}
}

func NewCloudflare(opts ...OptFunc) (DNSManager, error) {
	c := &Cloudflare{}

	for _, opt := range opts {
		opt(c)
	}

	if c.baseDomain == "" {
		return nil, fmt.Errorf("baseDNS must be set")
	}

	if c.ipV4 == "" {
		return nil, fmt.Errorf("ipV4 must be set")
	}

	if c.zoneID == "" {
		return nil, fmt.Errorf("zoneID must be set")
	}

	if c.token == "" {
		return nil, fmt.Errorf("token must be set")
	}

	return c, nil
}

type cloudflareDNSCreateRequest struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
	Comment string `json:"comment"`
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

func (c *Cloudflare) CreateDNSRecord(ctx context.Context, ttl time.Duration, record *Record) (*DNSRecord, error) {
	sub := strings.Split(c.baseDomain, ".")[0]

	body := cloudflareDNSCreateRequest{
		Type:    record.Type,
		Name:    fmt.Sprintf("%s.%s", record.Name, sub),
		Content: c.ipV4,
		TTL:     record.TTL,
		Proxied: record.Proxied,
		Comment: "created via wormhole",
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", c.zoneID),
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, ErrFailedToCreateNewDNSRecord
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var parsed cloudflareDNSCreateResponse
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		return nil, errors.Join(ErrFailedToCreateNewDNSRecord, err)
	}

	if !parsed.Success {
		for _, err := range parsed.Errors {
			if strings.Contains(err.Message, "record already exists") {
				return nil, ErrRecordAlreadyExists
			}
		}

		return nil, ErrFailedToCreateNewDNSRecord
	}

	return &DNSRecord{
		Meta: record,
		ID:   parsed.Result.ID,
		TTL:  ttl,
	}, nil
}

func (c *Cloudflare) DeleteDNSRecord(ctx context.Context, recordID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", c.zoneID, recordID),
		nil,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFailedToDeleteDNSRecord, err)
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("%w: failed to read response body: %v", ErrFailedToDeleteDNSRecord, readErr)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: %s", ErrFailedToDeleteDNSRecord, string(data))
	}

	return nil
}

func (c *Cloudflare) BaseDomain() string { return c.baseDomain }
func (c *Cloudflare) IPV4() string       { return c.ipV4 }
