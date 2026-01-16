package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// BasicAuth implements HTTP basic auth with bcrypt password hashing.
type BasicAuth struct {
	username string
	password string
	realm    string
}

func NewBasicAuth(username, password string) (*BasicAuth, error) {
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}
	if password == "" {
		return nil, errors.New("password cannot be empty")
	}

	return &BasicAuth{
		username: username,
		password: password,
		realm:    "Wormhole Tunnel",
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

	username := credentials[0]
	password := credentials[1]

	usernameMatch := subtle.ConstantTimeCompare(
		[]byte(username),
		[]byte(b.username),
	) == 1

	passwordMatch := subtle.ConstantTimeCompare(
		[]byte(password),
		[]byte(b.password),
	) == 1

	authenticated := usernameMatch && passwordMatch

	return authenticated
}

func (b *BasicAuth) SendChallenge(conn net.Conn) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Required</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        * {
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            padding: 1rem;
            background: #0f172a;
            color: #e2e8f0;
        }
        .container {
            text-align: center;
            max-width: 500px;
            width: 100%;
            padding: 2rem 1.5rem;
        }
        .error-code {
            font-size: clamp(4rem, 15vw, 6rem);
            font-weight: 700;
            margin-bottom: 1rem;
            opacity: 0.9;
            color: #64748b;
        }
        h1 {
            color: #f1f5f9;
            margin: 0 0 0.5rem 0;
            font-size: clamp(1.5rem, 5vw, 2rem);
            font-weight: 600;
        }
        p {
            color: #94a3b8;
            line-height: 1.6;
            margin: 0.5rem 0;
            font-size: clamp(0.938rem, 3vw, 1rem);
        }
        .code {
            display: inline-block;
            margin-top: 1rem;
            padding: 0.5rem 1rem;
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 4px;
            font-family: 'Courier New', monospace;
            font-size: clamp(0.813rem, 2.5vw, 0.875rem);
            color: #cbd5e1;
            word-break: break-all;
            max-width: 100%;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="error-code">401</div>
        <h1>Authentication Required</h1>
        <p>This tunnel requires valid credentials to access.</p>
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
		b.realm, len(html), html,
	)

	if _, err := conn.Write([]byte(response)); err != nil {
		log.Error().Err(err).Msg("failed to send basic auth challenge")
	}
}
