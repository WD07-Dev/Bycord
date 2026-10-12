package mobile

import (
	"bycord/core"
	"context"
	"encoding/json"
	"errors"
	"time"
)

type result struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Country   string `json:"country"`
	LatencyMS int64  `json:"latency_ms"`
}

func CountryISO() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return core.CountryISO(ctx)
}

func FindProxy(excludedCountries string) (string, error) {
	excluded := core.ParseExcluded(excludedCountries)
	if len(excluded) == 0 {
		return "", errors.New("enter at least one two-letter excluded country code")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	choice, err := core.Resolve(ctx, excluded, nil)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(result{
		Host:      choice.Endpoint.Host,
		Port:      choice.Endpoint.Port,
		Country:   choice.Country,
		LatencyMS: choice.Latency.Milliseconds(),
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func ValidExcluded(raw string) bool {
	return len(core.ParseExcluded(raw)) > 0
}
