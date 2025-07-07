package wormhole

import "time"

type Record struct {
	Type    string
	Name    string
	Content string
	TTL     int
	Proxied bool
}

type DNSRecord struct {
	Meta    Record
	ID      string
	Expires time.Duration
}
