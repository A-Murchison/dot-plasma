package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type applyReloadMode string

const (
	applyReloadNone        applyReloadMode = "none"
	applyReloadPlasmashell applyReloadMode = "plasmashell"
)

var (
	lookPath            = exec.LookPath
	runCommand          = runExternalCommand
	startCommand        = startExternalCommand
	quietAfterStop      = func() { time.Sleep(2 * time.Second) }
	currentEffectiveUID = os.Geteuid
)

func parseApplyReloadMode(value string) (applyReloadMode, error) {
	switch applyReloadMode(value) {
	case applyReloadNone, applyReloadPlasmashell:
		return applyReloadMode(value), nil
	default:
		return "", fmt.Errorf("unsupported reload mode %q; use none or plasmashell", value)
	}
}

func stopPlasmashellForApply(ctx context.Context) error {
	if currentEffectiveUID() == 0 {
		return fmt.Errorf("refusing to reload plasmashell as root")
	}

	commands := []struct {
		name string
		args []string
	}{
		{name: "kquitapp6", args: []string{"plasmashell"}},
		{name: "kquitapp5", args: []string{"plasmashell"}},
	}
	var attempted []string
	var failures []string
	for _, command := range commands {
		path, err := lookPath(command.name)
		if err != nil {
			continue
		}
		attempted = append(attempted, command.name)
		if err := runCommand(ctx, path, command.args...); err != nil {
			failures = append(failures, err.Error())
			continue
		}
		quietAfterStop()
		return nil
	}
	if len(attempted) == 0 {
		return fmt.Errorf("cannot stop plasmashell: neither kquitapp6 nor kquitapp5 was found")
	}
	return fmt.Errorf("cannot stop plasmashell: %s", strings.Join(failures, "; "))
}

func startPlasmashellAfterApply(ctx context.Context) error {
	if currentEffectiveUID() == 0 {
		return fmt.Errorf("refusing to reload plasmashell as root")
	}

	commands := []struct {
		name string
		args []string
		wait bool
	}{
		{name: "kstart", args: []string{"plasmashell"}, wait: true},
		{name: "kstart5", args: []string{"plasmashell"}, wait: true},
		{name: "plasmashell", wait: false},
	}
	var attempted []string
	var failures []string
	for _, command := range commands {
		path, err := lookPath(command.name)
		if err != nil {
			continue
		}
		attempted = append(attempted, command.name)
		if command.wait {
			if err := runCommand(ctx, path, command.args...); err != nil {
				failures = append(failures, err.Error())
				continue
			}
			return nil
		}
		if err := startCommand(ctx, path, command.args...); err != nil {
			failures = append(failures, err.Error())
			continue
		}
		return nil
	}
	if len(attempted) == 0 {
		return fmt.Errorf("cannot start plasmashell: none of kstart, kstart5, or plasmashell was found")
	}
	return fmt.Errorf("cannot start plasmashell: %s", strings.Join(failures, "; "))
}

func runExternalCommand(ctx context.Context, name string, args ...string) error {
	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(cmdCtx, name, args...).CombinedOutput()
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		return fmt.Errorf("run %s: %w", name, err)
	}
	return fmt.Errorf("run %s: %w: %s", name, err, message)
}

func startExternalCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}
	return nil
}
