package dnsmanager

import "time"

const (
	RecordTypeA = "A"
)

type Record struct {
	Type    string
	Name    string
	Content string
	TTL     int
	Proxied bool
}

type DNSRecord struct {
	Meta *Record
	ID   string
	TTL  time.Duration // defines how long the tunnel will live
}
