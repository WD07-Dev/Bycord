package core

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	discordHost      = "discord.com"
	geoHost          = "ifconfig.co"
	maxResponseBytes = 64 * 1024
)

type Endpoint struct {
	Scheme, Host string
	Port         int
}

func (e Endpoint) String() string  { return fmt.Sprintf("%s://%s:%d", e.Scheme, e.Host, e.Port) }
func (e Endpoint) Address() string { return net.JoinHostPort(e.Host, strconv.Itoa(e.Port)) }

var proxyPattern = regexp.MustCompile(`^(socks5|https?)://([a-z0-9.-]{1,253}):(\d{1,5})$`)

func ParseProxy(raw string) (Endpoint, bool) {
	m := proxyPattern.FindStringSubmatch(strings.TrimSpace(strings.ToLower(raw)))
	if m == nil {
		return Endpoint{}, false
	}
	port, err := strconv.Atoi(m[3])
	if err != nil || port < 1 || port > 65535 {
		return Endpoint{}, false
	}
	return Endpoint{Scheme: m[1], Host: m[2], Port: port}, true
}

func OpenTunnel(endpoint Endpoint, host string, port int, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", endpoint.Address(), timeout)
	if err != nil {
		return nil, err
	}
	if err = conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		conn.Close()
		return nil, err
	}
	if endpoint.Scheme == "socks5" {
		err = negotiateSOCKS5(conn, host, port)
	} else {
		err = negotiateCONNECT(conn, host, port)
	}
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err = conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func negotiateSOCKS5(conn net.Conn, host string, port int) error {
	if len(host) > 255 {
		return errors.New("host is too long for SOCKS5")
	}
	if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
		return err
	}
	greeting := make([]byte, 2)
	if _, err := io.ReadFull(conn, greeting); err != nil {
		return err
	}
	if greeting[0] != 5 || greeting[1] != 0 {
		return fmt.Errorf("proxy requires authentication (method 0x%02x)", greeting[1])
	}
	request := append([]byte{5, 1, 0, 3, byte(len(host))}, []byte(host)...)
	request = append(request, byte(port>>8), byte(port))
	if _, err := conn.Write(request); err != nil {
		return err
	}
	head := make([]byte, 4)
	if _, err := io.ReadFull(conn, head); err != nil {
		return err
	}
	if head[1] != 0 {
		return fmt.Errorf("proxy rejected destination (code 0x%02x)", head[1])
	}
	rest := 0
	switch head[3] {
	case 1:
		rest = 6
	case 4:
		rest = 18
	case 3:
		length := []byte{0}
		if _, err := io.ReadFull(conn, length); err != nil {
			return err
		}
		rest = int(length[0]) + 2
	default:
		return fmt.Errorf("unknown SOCKS5 address type 0x%02x", head[3])
	}
	_, err := io.ReadFull(conn, make([]byte, rest))
	return err
}

func negotiateCONNECT(conn net.Conn, host string, port int) error {
	target := net.JoinHostPort(host, strconv.Itoa(port))
	if _, err := fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target); err != nil {
		return err
	}
	var head []byte
	buf := make([]byte, 1)
	for !strings.HasSuffix(string(head), "\r\n\r\n") {
		if _, err := io.ReadFull(conn, buf); err != nil {
			return err
		}
		head = append(head, buf[0])
		if len(head) > 8192 {
			return errors.New("proxy response header is too long")
		}
	}
	status := string(head)
	if !strings.HasPrefix(status, "HTTP/1.1 200") && !strings.HasPrefix(status, "HTTP/1.0 200") {
		line, _, _ := strings.Cut(status, "\r\n")
		return fmt.Errorf("CONNECT rejected: %s", line)
	}
	return nil
}

func readTLS(conn net.Conn, host, path string, timeout time.Duration) (string, error) {
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}
	client := tls.Client(conn, &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
	if err := client.Handshake(); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(client, "GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s/%s\r\nConnection: close\r\n\r\n", path, host, AppName, Version); err != nil {
		return "", err
	}
	body, err := io.ReadAll(io.LimitReader(client, maxResponseBytes))
	if len(body) == 0 && err != nil {
		return "", err
	}
	return string(body), nil
}

func Probe(endpoint Endpoint, timeout time.Duration) (time.Duration, error) {
	started := time.Now()
	conn, err := OpenTunnel(endpoint, discordHost, 443, timeout)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	response, err := readTLS(conn, discordHost, "/api/v9/gateway", timeout)
	if err != nil {
		return 0, err
	}
	if !strings.HasPrefix(response, "HTTP/1.1 200") && !strings.HasPrefix(response, "HTTP/1.0 200") {
		line, _, _ := strings.Cut(response, "\r\n")
		return 0, fmt.Errorf("unexpected Discord response: %s", line)
	}
	return time.Since(started), nil
}

var countryPattern = regexp.MustCompile(`"country_iso"\s*:\s*"([A-Za-z]{2})"|"country(?:Code)?"\s*:\s*"([A-Za-z]{2})"`)

func ExitCountry(endpoint Endpoint, timeout time.Duration) (string, error) {
	conn, err := OpenTunnel(endpoint, geoHost, 443, timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	response, err := readTLS(conn, geoHost, "/json", timeout)
	if err != nil {
		return "", err
	}
	m := countryPattern.FindStringSubmatch(response)
	if m == nil {
		return "", errors.New("country is missing from geolocation response")
	}
	if m[1] != "" {
		return strings.ToUpper(m[1]), nil
	}
	return strings.ToUpper(m[2]), nil
}
