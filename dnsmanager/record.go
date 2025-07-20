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
	Meta    *Record
	ID      string
	Expires time.Duration
}
