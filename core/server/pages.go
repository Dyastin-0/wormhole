package server

import (
	"fmt"
	"net/http"
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

const favicon = `
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32">
  <defs>
    <clipPath id="c">
      <circle cx="16" cy="16" r="15"/>
    </clipPath>
  </defs>
  <circle cx="16" cy="16" r="15" fill="#0a0a0a"/>
  <g clip-path="url(#c)">
    <ellipse cx="13" cy="16" rx="12" ry="14" fill="none" stroke="#F07178" stroke-width="4"/>
    <ellipse cx="16" cy="16" rx="8"  ry="10" fill="none" stroke="#F07178" stroke-width="3.5"/>
    <ellipse cx="18" cy="16" rx="4.5" ry="6" fill="none" stroke="#F07178" stroke-width="3"/>
    <ellipse cx="20" cy="16" rx="2"  ry="2.5" fill="none" stroke="#F07178" stroke-width="2"/>
    <circle  cx="20" cy="16" r="1"  fill="#F07178"/>
  </g>
</svg>`

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>Wormhole</title>
<link rel="icon" type="image/svg+xml" href="/favicon.svg">
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
    <a target="_blank" href="https://github.com/Dyastin-0/wormhole">Documentation</a>
  </div>
</div>
</body>
</html>`, baseCSS, proto.VERSION)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func (s *Server) faviconHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	fmt.Fprint(w, favicon)
}

func (s *Server) forbidden(w http.ResponseWriter, r *http.Request) {
	sni := r.Host
	name := strings.Split(sni, ".")[0]
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>403 Forbidden</title>
<link rel="icon" type="image/svg+xml" href="https://%s/favicon.svg">
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
    <a target="_blank" href="https://github.com/Dyastin-0/wormhole">Documentation</a>
  </div>
</div>
</body>
</html>`, s.domain, baseCSS, sni, name, proto.VERSION)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprint(w, html)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	sni := r.Host
	name := strings.Split(sni, ".")[0]
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>404 Not Found</title>
<link rel="icon" type="image/svg+xml" href="https://%s/favicon.svg">
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
    <a target="_blank" href="https://github.com/Dyastin-0/wormhole">Documentation</a>
  </div>
</div>
</body>
</html>`, s.domain, baseCSS, sni, name, proto.VERSION)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, html)
}

func (s *Server) unauthorizedBasic(realm string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>401 Unauthorized</title>
<link rel="icon" type="image/svg+xml" href="https://%s/favicon.svg">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
%s
</head>
<body>
<div class="container">
  <span class="brand">Wormhole /// Gateway</span>

  <div class="header" style="border-bottom: none; padding-bottom: 0;">
    <h1>Authentication Required</h1>
    <div class="description">
      This tunnel is private. Please enter your credentials to proceed.
    </div>
  </div>

  <div class="section">
    <span class="label">Security Realm</span>
    <div class="code" style="user-select: none;">%s</div>
  </div>

  <div class="footer">
    <span>v%s</span>
    <a target="_blank" href="https://github.com/Dyastin-0/wormhole">Documentation</a>
  </div>
</div>
</body>
</html>`, s.domain, baseCSS, realm, proto.VERSION)

		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", realm))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, html)
	}
}

func (s *Server) unauthorizedBearer(realm string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<title>401 Unauthorized</title>
<link rel="icon" type="image/svg+xml" href="https://%s/favicon.svg">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
%s
</head>
<body>
<div class="container">
  <span class="brand">Wormhole /// Gateway</span>

  <div class="header" style="border-bottom: none; padding-bottom: 0;">
    <h1>Bearer Token Required</h1>
    <div class="description">
      This tunnel requires a valid Bearer token to access.
    </div>
  </div>

  <div class="section">
    <span class="label">Required Header</span>
    <div class="code">Authorization: Bearer &lt;your-token&gt;</div>
  </div>

  <div class="section">
    <span class="label">Security Realm</span>
    <div class="code" style="user-select: none; opacity: 0.7;">%s</div>
  </div>

  <div class="footer">
    <span>v%s</span>
    <a target="_blank" href="https://github.com/Dyastin-0/wormhole">Documentation</a>
  </div>
</div>
</body>
</html>`, s.domain, baseCSS, realm, proto.VERSION)

		w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer realm=\"%s\"", realm))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, html)
	}
}
