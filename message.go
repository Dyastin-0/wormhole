package wormhole

type message struct {
	Action      string `json:"action"`
	Status      int    `json:"status"`
	Err         string `json:"err"`
	APIKey      string `json:"api_key"`
	TunnelProto string `json:"proto"`
	TunnelName  string `json:"name"`
	TunnelID    string `json:"tunnel_id"`
}
