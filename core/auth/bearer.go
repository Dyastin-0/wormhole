package auth

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// BearerAuth implements bearer token auth.
type BearerAuth struct {
	token string
	realm string
}

func NewBearerAuth(token string) (*BearerAuth, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	return &BearerAuth{
		token: token,
		realm: "Wormhole Tunnel",
	}, nil
}

func (b *BearerAuth) Authenticate(req *http.Request) bool {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		log.Debug().Msg("no authorization header")
		return false
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		log.Debug().Msg("not bearer auth")
		return false
	}

	token := strings.TrimSpace(auth[7:])

	return subtle.ConstantTimeCompare(
		[]byte(token),
		[]byte(b.token),
	) == 1
}

func (b *BearerAuth) SendChallenge(conn net.Conn) {
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
        <h1>Bearer Token Required</h1>
        <p>This tunnel requires a valid Bearer token to access.</p>
        <div class="code">Authorization: Bearer &lt;token&gt;</div>
    </div>
</body>
</html>`

	response := fmt.Sprintf(
		"HTTP/1.1 401 Unauthorized\r\n"+
			"WWW-Authenticate: Bearer realm=\"%s\"\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		b.realm, len(html), html,
	)

	if _, err := conn.Write([]byte(response)); err != nil {
		log.Error().Err(err).Msg("failed to send bearer auth challenge")
	}
}
