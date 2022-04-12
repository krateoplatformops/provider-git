package req

import (
	"context"
	"net/http"
	"testing"
)

func TestUserAgentTransport(t *testing.T) {
	// Wrap an existing transport or use nil for http.DefaultTransport
	baseTrans := http.DefaultClient.Transport
	trans := UserAgentTransport(baseTrans, "my-user/agent")

	var headers postman
	err := Get().
		Url("https://postman-echo.com/get").
		Transport(trans).
		ToJSON(&headers).
		Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if h := headers.Headers["user-agent"]; h != "my-user/agent" {
		t.Fatalf("bad user agent: %q", h)
	}
}
