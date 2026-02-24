package auth

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicAuth(t *testing.T) {
	auth, _ := NewBasicAuth("user", "pass")
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:pass")))
	require.True(t, auth.Authenticate(req))
}

func TestBearerAuth(t *testing.T) {
	auth, _ := NewBearerAuth("secret-token")
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	require.True(t, auth.Authenticate(req))
}

func TestNoAuth(t *testing.T) {
	auth := &NoAuth{}
	req, _ := http.NewRequest("GET", "/", nil)
	require.True(t, auth.Authenticate(req))
}
