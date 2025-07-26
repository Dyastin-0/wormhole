package wormhole

type message struct {
	Message      string `json:"action"`
	Status       int    `json:"status"`
	Err          string `json:"err"`
	APIKey       string `json:"api_key"`
	TunnelProto  string `json:"proto"`
	TunnelName   string `json:"name"`
	TunnelID     string `json:"tunnel_id"`
	TunnelDomain string `json:"tunne_domain"`
}
