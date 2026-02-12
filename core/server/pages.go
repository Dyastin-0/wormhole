package server

import (
	"fmt"
	"net"
	"strings"

	"github.com/Dyastin-0/wormhole/core/proto"
)

const baseCSS = `
<style>
:root {
  --bg: #f4f4f6;
  --surface: #ffffff;
  --border: #bdbdbd;
  --text: #000000;
  --subtext: #4a4a4e;
  --code-bg: #f1f1f1;
  --accent: #F07178; 
  --mono-font: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

@media (prefers-color-scheme: dark) {
  :root {
    --bg: #050505;
    --surface: #0a0a0a;
    --border: #262626;
    --text: #ededed;
    --subtext: #888888;
    --code-bg: #111111;
    --accent: #F07178; 
  }
}

* { box-sizing: border-box; }

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  background: var(--bg);
  color: var(--text);
  line-height: 1.5;
  margin: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 20px;
}

.container {
  width: 100%;
  max-width: 600px;
  background: var(--surface);
  border: 1px solid var(--border);
  padding: 40px;
}

.header {
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border);
}

.brand {
  font-family: var(--mono-font);
  font-weight: 800;
  font-size: 14px;
  letter-spacing: -0.5px;
  color: var(--accent);
  text-transform: uppercase;
  margin-bottom: 8px;
  display: block;
}

h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.5px;
}

.description {
  color: var(--subtext);
  margin-top: 8px;
  font-size: 15px;
}

.section {
  margin-top: 24px;
}

.label {
  font-size: 12px;
  font-weight: 600;
  color: var(--subtext);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
  display: block;
}

.code {
  font-family: var(--mono-font);
  background: var(--code-bg);
  border: 1px solid var(--border);
  padding: 12px 16px;
  font-size: 13px;
  color: var(--text);
  user-select: all; 
  cursor: text;
  word-break: break-all;
  position: relative;
}

.code::before {
  content: "$";
  color: var(--subtext);
  margin-right: 8px;
  user-select: none; 
}

.footer {
  margin-top: 40px;
  padding-top: 20px;
  border-top: 1px solid var(--border);
  font-size: 13px;
  color: var(--subtext);
  display: flex;
  justify-content: space-between;
}

a {
  color: var(--text);
  text-decoration: none;
  border-bottom: 1px solid var(--border);
  transition: border-color 0.2s;
}

a:hover {
  border-bottom-color: var(--accent);
}

.status-badge {
  display: inline-block;
  padding: 2px 8px;
	color: var(--accent);
  font-size: 11px;
  font-weight: 600;
  border: 1px solid var(--border);
  margin-bottom: 16px;
  font-family: var(--mono-font);
}
</style>
`

func (s *Server) writeHomePage(conn net.Conn) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>Wormhole</title>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
%s
</head>
<body>
<div class="container">
  <div class="header">
    <span class="brand">Wormhole /// Gateway</span>
    <h1>Your localhost, live.</h1>
    <div class="description">Secure tunnels for development, game servers, and more.</div>
  </div>

  <div class="section">
    <span class="label">Installation</span>
    <div class="code">go install github.com/Dyastin-0/wormhole@latest</div>
  </div>

  <div class="section">
    <span class="label">Usage</span>
    <div class="code">wormhole http --target :3000</div>
  </div>

  <div class="footer">
    <span>v%s</span>
    <a href="https://github.com/Dyastin-0/wormhole">Documentation</a>
  </div>
</div>
</body>
</html>`, baseCSS, proto.VERSION)

	writeHTTP(conn, 200, html)
}

func (s *Server) writeForbidden(conn net.Conn, sni string) {
	name := strings.Split(sni, ".")[0]
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>403 Forbidden</title>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
%s
</head>
<body>
<div class="container">
  <span class="brand">Wormhole /// Gateway</span>
  
  <div class="header" style="border-bottom: none; padding-bottom: 0;">
    <h1>Protocol Mismatch</h1>
    <div class="description">
       You are trying to access a TCP tunnel via HTTP.<br>
       The gateway cannot proxy this request.
    </div>
  </div>

  <div class="section">
    <span class="label">Target Tunnel</span>
    <div class="code" style="user-select: none;">%s</div>
  </div>

  <div class="section">
    <span class="label">Fix: Enable HTTP Mode</span>
    <div class="code">wormhole tcp --name %s --allow-http</div>
  </div>

  <div class="footer">
    <span>v%s</span>
    <a href="https://github.com/Dyastin-0/wormhole">Documentation</a>
  </div>
</div>
</body>
</html>`, baseCSS, sni, name, proto.VERSION)

	writeHTTP(conn, 403, html)
}

func (s *Server) writeNotFound(conn net.Conn, sni string) {
	name := strings.Split(sni, ".")[0]
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>404 Not Found</title>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
%s
</head>
<body>
<div class="container">
  <span class="brand">Wormhole /// Gateway</span>

  <div class="header" style="border-bottom: none; padding-bottom: 0;">
    <h1>Tunnel Not Active</h1>
    <div class="description">
       No active session found for this endpoint.
    </div>
  </div>

  <div class="section">
    <span class="label">Requested Host</span>
    <div class="code" style="user-select: none;">%s</div>
  </div>

  <div class="section">
    <span class="label">To Claim This Domain</span>
    <div class="code">wormhole http --name %s --target :3000</div>
  </div>

  <div class="footer">
    <span>v%s</span>
    <a href="https://github.com/Dyastin-0/wormhole">Documentation</a>
  </div>
</div>
</body>
</html>`, baseCSS, sni, name, proto.VERSION)

	writeHTTP(conn, 404, html)
}

func writeHTTP(conn net.Conn, status int, body string) {
	statusText := map[int]string{
		200: "OK",
		403: "Forbidden",
		404: "Not Found",
	}[status]

	resp := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n\r\n%s",
		status, statusText, len(body), body,
	)
	conn.Write([]byte(resp))
}
