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
    <title>401 Unauthorized</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        .container {
            text-align: center;
            padding: 3rem;
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 400px;
        }
        h1 {
            color: #333;
            margin: 0 0 0.5rem 0;
            font-size: 3rem;
        }
        h2 {
            color: #667eea;
            margin: 0 0 1.5rem 0;
            font-weight: 600;
        }
        p {
            color: #666;
            line-height: 1.6;
            margin: 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>401</h1>
        <h2>Unauthorized</h2>
        <p>This Wormhole tunnel requires authentication.</p>
        <p>Please provide valid credentials to continue.</p>
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
