package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const freeProxyAPI = "https://api.proxyscrape.com/v4/free-proxy-list/get?request=display_proxies&protocol=socks5&proxy_format=protocolipport&format=json&timeout=1500"

type Choice struct {
	Endpoint Endpoint
	Latency  time.Duration
	Country  string
}

type Progress func(done, total int)

func ParseExcluded(raw string) map[string]bool {
	out := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		code := strings.ToUpper(strings.TrimSpace(part))
		if len(code) == 2 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' {
			out[code] = true
		}
	}
	return out
}

func Validate(endpoint Endpoint, excluded map[string]bool, timeout time.Duration) (Choice, error) {
	latency, err := Probe(endpoint, timeout)
	if err != nil {
		return Choice{}, err
	}
	country, err := ExitCountry(endpoint, timeout)
	if err != nil {
		return Choice{}, err
	}
	if excluded[country] {
		return Choice{}, fmt.Errorf("proxy exits through excluded country %s", country)
	}
	return Choice{Endpoint: endpoint, Latency: latency, Country: country}, nil
}

type listEntry struct {
	Proxy   string  `json:"proxy"`
	Alive   bool    `json:"alive"`
	Uptime  float64 `json:"uptime"`
	Timeout float64 `json:"timeout"`
	IPData  struct {
		CountryCode string `json:"countryCode"`
	} `json:"ip_data"`
}

func candidates(ctx context.Context, excluded map[string]bool) ([]Endpoint, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, freeProxyAPI, nil)
	if err != nil {
		return nil, err
	}
	res, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy list returned HTTP %d", res.StatusCode)
	}
	var payload struct {
		Proxies []listEntry `json:"proxies"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1024*1024)).Decode(&payload); err != nil {
		return nil, err
	}
	sort.SliceStable(payload.Proxies, func(i, j int) bool { return payload.Proxies[i].Uptime > payload.Proxies[j].Uptime })
	result := make([]Endpoint, 0, 40)
	for _, item := range payload.Proxies {
		endpoint, ok := ParseProxy(item.Proxy)
		if !ok || !item.Alive || item.Uptime < 90 || item.Timeout > 1500 || excluded[strings.ToUpper(item.IPData.CountryCode)] {
			continue
		}
		result = append(result, endpoint)
		if len(result) == 40 {
			break
		}
	}
	if len(result) == 0 {
		return nil, errors.New("no usable proxy candidates")
	}
	return result, nil
}

func Resolve(ctx context.Context, excluded map[string]bool, progress Progress) (Choice, error) {
	pool, err := candidates(ctx, excluded)
	if err != nil {
		return Choice{}, err
	}
	type result struct {
		choice Choice
		err    error
	}
	jobs := make(chan Endpoint)
	results := make(chan result)
	var wg sync.WaitGroup
	workers := 10
	if len(pool) < workers {
		workers = len(pool)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for endpoint := range jobs {
				choice, err := Validate(endpoint, excluded, 6*time.Second)
				results <- result{choice, err}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, endpoint := range pool {
			select {
			case jobs <- endpoint:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()
	var valid []Choice
	done := 0
	for result := range results {
		done++
		if progress != nil {
			progress(done, len(pool))
		}
		if result.err == nil {
			valid = append(valid, result.choice)
		}
	}
	if err := ctx.Err(); err != nil {
		return Choice{}, err
	}
	if len(valid) == 0 {
		return Choice{}, errors.New("no proxy passed Discord and country validation")
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].Latency < valid[j].Latency })
	return valid[0], nil
}
