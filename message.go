package wormhole

type message struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Status int    `json:"status"`
}
