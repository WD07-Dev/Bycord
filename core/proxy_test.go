package core

import "testing"

func TestParseProxy(t *testing.T) {
	endpoint, ok := ParseProxy("SOCKS5://127.0.0.1:9050")
	if !ok || endpoint.Scheme != "socks5" || endpoint.Port != 9050 {
		t.Fatalf("unexpected endpoint: %#v, %v", endpoint, ok)
	}
	for _, bad := range []string{"", "ftp://host:21", "http://host:0", "host:80"} {
		if _, ok := ParseProxy(bad); ok {
			t.Errorf("accepted invalid proxy %q", bad)
		}
	}
}

func TestParseExcluded(t *testing.T) {
	got := ParseExcluded("br, US, invalid, 1A")
	if !got["BR"] || !got["US"] || len(got) != 2 {
		t.Fatalf("unexpected exclusions: %#v", got)
	}
}
