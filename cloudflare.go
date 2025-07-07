package wormhole

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go/v4"
	cloudflare_dns "github.com/cloudflare/cloudflare-go/v4/dns"
	cloudflare_option "github.com/cloudflare/cloudflare-go/v4/option"
)

type CloudflareAPI struct {
	api  *cloudflare.Client
	zone string
}

// cloudflare.NewClient() automatically sets default options read from env
// (CLOUDFLARE_API_KEY, CLOUDFLARE_API_USER_SERVICE_KEY,
// CLOUDFLARE_API_TOKEN, CLOUDFLARE_EMAIL, CLOUDFLARE_BASE_URL).
func NewCloudflareAPI(zone string) DNSAPI {
	return &CloudflareAPI{
		api:  cloudflare.NewClient(),
		zone: zone,
	}
}

func NewCloudflareAPIWithOpts(zone string, opts ...cloudflare_option.RequestOption) DNSAPI {
	return &CloudflareAPI{
		api:  cloudflare.NewClient(opts...),
		zone: zone,
	}
}

func (cfc *CloudflareAPI) CreateDNSRecord(ctx context.Context, expires time.Duration, record Record) (*DNSRecord, error) {
	res, err := cfc.api.DNS.Records.New(
		ctx,
		cloudflare_dns.RecordNewParams{
			ZoneID: cloudflare.F(cfc.zone),
			Body: cloudflare_dns.ARecordParam{
				Name:    cloudflare.F(record.Name),
				TTL:     cloudflare.F(cloudflare_dns.TTL(record.TTL)),
				Type:    cloudflare.F(cloudflare_dns.ARecordType(record.Type)),
				Proxied: cloudflare.F(record.Proxied),
				Content: cloudflare.F(record.Content),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToCreateNewDNSRecord, err)
	}

	dns := &DNSRecord{
		Meta:    record,
		ID:      res.ID,
		Expires: expires,
	}

	return dns, nil
}

func (cfc *CloudflareAPI) DeleteDNSRecord(ctx context.Context, id string) error {
	_, err := cfc.api.DNS.Records.Delete(
		ctx,
		id,
		cloudflare_dns.RecordDeleteParams{
			ZoneID: cloudflare.F(cfc.zone),
		},
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToDeleteDNSRecord, err)
	}

	return nil
}
