package server

import (
	"fmt"
	"net"
	"strings"
)

func (s *Server) writeHomePage(conn net.Conn) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Wormhole</title>
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
            width: 100%%;
            padding: 2rem 1.5rem;
        }
        .title {
            font-size: clamp(2rem, 8vw, 3rem);
            font-weight: 700;
            margin-bottom: 0.5rem;
            color: #f1f5f9;
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
            max-width: 100%%;
        }
        .hint {
            margin-top: 1.5rem;
            padding-top: 1.5rem;
            border-top: 1px solid #334155;
            color: #64748b;
            font-size: clamp(0.813rem, 2.5vw, 0.875rem);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="title">Wormhole</div>
        <p>Expose local services to the internet.</p>
        <div class="hint">
            <p>Install with:</p>
            <p class="code">go install github.com/Dyastin-0/wormhole@latest</p>
        </div>
        <div class="hint">
            <p>Create a tunnel:</p>
						<p class="code">wormhole http --name myapp --target-address :3000</p>
        </div>
        <div class="hint">
            <p>Visit <a href="https://github.com/Dyastin-0/wormhole" target="_blank" style="color: #60a5fa; text-decoration: none;">github.com/Dyastin-0/wormhole</a> for documentation.</p>
        </div>
    </div>
</body>
</html>`

	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		len(html), html,
	)
	conn.Write([]byte(response))
}

func (s *Server) writeForbidden(conn net.Conn, sni string) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Forbidden</title>
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
            width: 100%%;
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
            max-width: 100%%;
        }
        .hint {
            margin-top: 1.5rem;
            padding-top: 1.5rem;
            border-top: 1px solid #334155;
            color: #64748b;
            font-size: clamp(0.813rem, 2.5vw, 0.875rem);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="error-code">403</div>
        <h1>Forbidden</h1>
        <p>HTTP access disabled for this TCP tunnel.</p>
        <div class="code">%s</div>
        <div class="hint">
						<p>If you are the owner of this tunnel, run:</p>
            <p class="code">wormhole tcp --name %s --target-address :3000 --allow-http</p>
        </div>
    </div>
</body>
</html>`, sni, strings.Split(sni, ".")[0])

	response := fmt.Sprintf(
		"HTTP/1.1 403 Forbidden\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		len(html), html,
	)
	conn.Write([]byte(response))
}

func (s *Server) writeNoTunnel(conn net.Conn, sni string) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Tunnel Not Found</title>
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
            width: 100%%;
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
            max-width: 100%%;
        }
        .hint {
            margin-top: 1.5rem;
            padding-top: 1.5rem;
            border-top: 1px solid #334155;
            color: #64748b;
            font-size: clamp(0.813rem, 2.5vw, 0.875rem);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="error-code">404</div>
        <h1>Tunnel Not Found</h1>
        <p>No active tunnel is registered for this subdomain.</p>
        <div class="code">%s</div>
        <div class="hint">
            <p>Create a tunnel with:</p>
            <p class="code">wormhole http --name %s --target-address :3000</p>
        </div>
    </div>
</body>
</html>`, sni, strings.Split(sni, ".")[0])

	response := fmt.Sprintf(
		"HTTP/1.1 404 Not Found\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		len(html), html,
	)
	conn.Write([]byte(response))
}
