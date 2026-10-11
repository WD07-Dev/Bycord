package desktop

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestUIConnectAndQuit(t *testing.T) {
	urlReady := make(chan string, 1)
	done := make(chan error, 1)
	go func() {
		done <- Run(func(url string) error { urlReady <- url; return nil }, func(_ context.Context, request Request) (Result, error) {
			if request.Exclude != "BR" {
				t.Errorf("unexpected request: %#v", request)
			}
			return Result{Endpoint: "socks5://127.0.0.1:9050", Country: "DE", Latency: 42, Message: "started"}, nil
		})
	}()

	url := <-urlReady
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "Bycord") {
		t.Fatalf("unexpected page response: %d", response.StatusCode)
	}

	response, err = http.Post(url+"connect", "application/json", bytes.NewBufferString(`{"exclude":"BR"}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"country":"DE"`) {
		t.Fatalf("unexpected connect response: %d %s", response.StatusCode, body)
	}

	response, err = http.Post(url+"quit", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("UI server did not stop")
	}
}
