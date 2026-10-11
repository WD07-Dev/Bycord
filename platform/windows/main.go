package main

import (
	"bycord/core"
	"bycord/platform/desktop"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type channel struct {
	folder string
	binary string
	label  string
}

var channels = []channel{
	{"Discord", "Discord.exe", "Discord"},
	{"DiscordPTB", "DiscordPTB.exe", "Discord PTB"},
	{"DiscordCanary", "DiscordCanary.exe", "Discord Canary"},
}

func main() {
	_ = runUI()
}

func runUI() error {
	return desktop.Run(openBrowser, connect)
}

func openBrowser(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func connect(ctx context.Context, request desktop.Request) (desktop.Result, error) {
	choice, err := desktop.SelectProxy(ctx, request.Proxy, request.Exclude)
	if err != nil {
		return desktop.Result{}, err
	}
	message, err := launchDiscord(choice, request.Exe, nil)
	if err != nil {
		return desktop.Result{}, err
	}
	return desktop.Result{
		Endpoint: choice.Endpoint.String(),
		Country:  choice.Country,
		Latency:  choice.Latency.Milliseconds(),
		Message:  message,
	}, nil
}

func launchDiscord(choice core.Choice, executable string, extra []string) (string, error) {
	ch, binary, err := locate(executable)
	if err != nil {
		return "", err
	}
	if running(ch.binary) {
		if out, err := exec.Command("taskkill", "/IM", ch.binary, "/F", "/T").CombinedOutput(); err != nil {
			return "", fmt.Errorf("cannot stop Discord: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}
	args := []string{
		"--proxy-server=" + choice.Endpoint.String(),
		"--proxy-bypass-list=" + desktop.DiscordProxyBypass,
	}
	cmd := exec.Command(binary, append(args, extra...)...)
	cmd.Dir = filepath.Dir(binary)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("cannot start Discord: %w", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return fmt.Sprintf("%s started through Bycord (pid %d)", ch.label, pid), nil
}

func locate(explicit string) (channel, string, error) {
	if explicit != "" {
		return channel{"", filepath.Base(explicit), filepath.Base(explicit)}, explicit, nil
	}
	root := os.Getenv("LOCALAPPDATA")
	for _, ch := range channels {
		matches, _ := filepath.Glob(filepath.Join(root, ch.folder, "app-*", ch.binary))
		sort.Sort(sort.Reverse(sort.StringSlice(matches)))
		if len(matches) > 0 {
			return ch, matches[0], nil
		}
	}
	return channel{}, "", fmt.Errorf("Discord was not found under LOCALAPPDATA; use -exe")
}

func running(binary string) bool {
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq "+binary, "/NH", "/FO", "CSV").Output()
	return err == nil && strings.Contains(strings.ToLower(string(out)), strings.ToLower(binary))
}
