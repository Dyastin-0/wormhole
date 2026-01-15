package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// BasicAuth implements HTTP basic auth.
type BasicAuth struct {
	Username string
	Password string
	Realm    string
}

func NewBasicAuth(username, password string) (*BasicAuth, error) {
	if username == "" {
		return nil, errors.New("nil username")
	}

	if password == "" {
		return nil, errors.New("nil password")
	}

	return &BasicAuth{
		Username: username,
		Password: password,
		Realm:    "Wormhole Tunnel",
	}, nil
}

func (b *BasicAuth) Authenticate(req *http.Request) bool {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return false
	}

	credentials := strings.SplitN(string(payload), ":", 2)
	if len(credentials) != 2 {
		return false
	}

	authenticated := credentials[0] == b.Username && credentials[1] == b.Password

	if authenticated {
		log.Info().Str("user", credentials[0]).Msg("successful authentication")
	} else {
		log.Warn().Str("user", credentials[0]).Msg("failed authentication")
	}

	return authenticated
}

func (b *BasicAuth) SendChallenge(conn net.Conn) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Required</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: #0f172a;
            color: #e2e8f0;
        }
        .container {
            text-align: center;
            max-width: 500px;
            padding: 2rem;
        }
        .error-code {
            font-size: 6rem;
            font-weight: 700;
            margin-bottom: 1rem;
            opacity: 0.9;
            color: #64748b;
        }
        h1 {
            color: #f1f5f9;
            margin: 0 0 0.5rem 0;
            font-size: 2rem;
            font-weight: 600;
        }
        p {
            color: #94a3b8;
            line-height: 1.6;
            margin: 0.5rem 0;
            font-size: 1rem;
        }
        .code {
            display: inline-block;
            margin-top: 1rem;
            padding: 0.25rem 0.75rem;
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
            font-size: 0.875rem;
            color: #cbd5e1;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="error-code">401</div>
        <h1>Authentication Required</h1>
        <p>This tunnel requires basic authentication to access.</p>
        <div class="code">401 Unauthorized</div>
    </div>
</body>
</html>`

	response := fmt.Sprintf(
		"HTTP/1.1 401 Unauthorized\r\n"+
			"WWW-Authenticate: Basic realm=\"%s\"\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		b.Realm, len(html), html,
	)

	conn.Write([]byte(response))
}

func (b *BasicAuth) IsEnabled() bool {
	return b.Username != "" && b.Password != ""
}
