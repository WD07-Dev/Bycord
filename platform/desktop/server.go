package desktop

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"
)

//go:embed static/*
var embeddedFiles embed.FS

type Request struct {
	Proxy   string `json:"proxy"`
	Exclude string `json:"exclude"`
	Exe     string `json:"exe"`
}

type Result struct {
	Endpoint string `json:"endpoint"`
	Country  string `json:"country"`
	Latency  int64  `json:"latency"`
	Message  string `json:"message"`
}

type ConnectFunc func(context.Context, Request) (Result, error)

func Run(openBrowser func(string) error, connect ConnectFunc) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	base := "/" + hex.EncodeToString(tokenBytes) + "/"
	assets, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second}
	mux.Handle(base, http.StripPrefix(base, http.FileServer(http.FS(assets))))
	mux.HandleFunc(base+"connect", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		var request Request
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "잘못된 요청입니다."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
		defer cancel()
		result, err := connect(ctx, request)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	})
	mux.HandleFunc(base+"quit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = server.Shutdown(context.Background())
		}()
	})
	server.Handler = mux
	url := fmt.Sprintf("http://%s%s", listener.Addr(), base)
	if err := openBrowser(url); err != nil {
		return fmt.Errorf("cannot open browser: %w", err)
	}
	err = server.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
