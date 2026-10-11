package main

import (
	"bycord/core"
	"bycord/platform/desktop"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const discordFlatpakID = "com.discordapp.Discord"

type installation struct {
	command   string
	args      []string
	process   string
	flatpakID string
	fullMatch bool
}

func main() {
	_ = runUI()
}

func runUI() error {
	return desktop.Run(openBrowser, connect)
}

func openBrowser(url string) error {
	return exec.Command("xdg-open", url).Start()
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
	install, err := locate(executable)
	if err != nil {
		return "", err
	}
	if install.running() {
		if err := install.stop(); err != nil {
			return "", fmt.Errorf("cannot stop Discord: %w", err)
		}
	}
	args := append([]string{}, install.args...)
	args = append(args,
		"--proxy-server="+choice.Endpoint.String(),
		"--proxy-bypass-list="+desktop.DiscordProxyBypass,
	)
	cmd := exec.Command(install.command, append(args, extra...)...)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("cannot start Discord: %w", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return fmt.Sprintf("Discord started through Bycord (pid %d)", pid), nil
}

func locate(explicit string) (installation, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return installation{}, err
		}
		return executableInstall(explicit), nil
	}
	for _, name := range []string{"discord", "discord-ptb", "discord-canary"} {
		if path, err := exec.LookPath(name); err == nil {
			return executableInstall(path), nil
		}
	}
	if install, ok := flatpakInstall(); ok {
		return install, nil
	}
	if path, ok := findAppImage(); ok {
		return executableInstall(path), nil
	}
	return installation{}, fmt.Errorf("Discord was not found in PATH, Flatpak, or AppImage folders; use -exe")
}

func executableInstall(path string) installation {
	return installation{
		command:   path,
		process:   filepath.Base(path),
		fullMatch: strings.EqualFold(filepath.Ext(path), ".AppImage"),
	}
}

func flatpakInstall() (installation, bool) {
	flatpak, err := exec.LookPath("flatpak")
	if err != nil || exec.Command(flatpak, "info", discordFlatpakID).Run() != nil {
		return installation{}, false
	}
	return installation{
		command:   flatpak,
		args:      []string{"run", discordFlatpakID},
		flatpakID: discordFlatpakID,
	}, true
}

func findAppImage() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	directories := []string{
		home,
		filepath.Join(home, "Applications"),
		filepath.Join(home, "AppImages"),
		filepath.Join(home, "Downloads"),
	}
	var matches []string
	for _, directory := range directories {
		entries, err := os.ReadDir(directory)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := strings.ToLower(entry.Name())
			if entry.IsDir() || !strings.Contains(name, "discord") || !strings.HasSuffix(name, ".appimage") {
				continue
			}
			info, err := entry.Info()
			if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0 {
				matches = append(matches, filepath.Join(directory, entry.Name()))
			}
		}
	}
	if len(matches) == 0 {
		return "", false
	}
	sort.Strings(matches)
	return matches[0], true
}

func (install installation) running() bool {
	if install.flatpakID != "" {
		out, err := exec.Command(install.command, "ps", "--columns=application").Output()
		if err != nil {
			return false
		}
		for _, appID := range strings.Fields(string(out)) {
			if appID == install.flatpakID {
				return true
			}
		}
		return false
	}
	if install.fullMatch {
		return exec.Command("pgrep", "-f", regexp.QuoteMeta(install.command)).Run() == nil
	}
	return exec.Command("pgrep", "-x", install.process).Run() == nil
}

func (install installation) stop() error {
	if install.flatpakID != "" {
		return exec.Command(install.command, "kill", install.flatpakID).Run()
	}
	if install.fullMatch {
		return exec.Command("pkill", "-f", regexp.QuoteMeta(install.command)).Run()
	}
	return exec.Command("pkill", "-x", install.process).Run()
}