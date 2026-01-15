package auth

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// BearerAuth implements bearer token auth.
type BearerAuth struct {
	Token string
	Realm string
}

func NewBearerAuth(token string) (*BearerAuth, error) {
	if token == "" {
		return nil, errors.New("nil token")
	}

	return &BearerAuth{
		Token: token,
		Realm: "Wormhole Tunnel",
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
	authenticated := token == b.Token

	if authenticated {
		log.Info().Msg("successful bearer authentication")
	} else {
		log.Warn().Msg("failed bearer authentication")
	}

	return authenticated
}

func (b *BearerAuth) SendChallenge(conn net.Conn) {
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
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
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
            color: #f5576c;
            margin: 0 0 1.5rem 0;
            font-weight: 600;
        }
        p {
            color: #666;
            line-height: 1.6;
            margin: 0.5rem 0;
        }
        code {
            background: #f5f5f5;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>401</h1>
        <h2>Unauthorized</h2>
        <p>This tunnel requires a Bearer token.</p>
        <p>Include header: <code>Authorization: Bearer &lt;token&gt;</code></p>
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
		b.Realm, len(html), html,
	)

	conn.Write([]byte(response))
}

func (b *BearerAuth) IsEnabled() bool {
	return b.Token != ""
}
