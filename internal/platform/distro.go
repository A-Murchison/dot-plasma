package platform

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var ErrDistroUnavailable = errors.New("distro unavailable")

func DetectDistro() (string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrDistroUnavailable
		}
		return "", fmt.Errorf("read /etc/os-release: %w", err)
	}
	distro := ParseOSReleaseDistro(string(data))
	if distro == "" {
		return "", ErrDistroUnavailable
	}
	return distro, nil
}

func ParseOSReleaseDistro(input string) string {
	values := parseOSRelease(input)
	if pretty := strings.TrimSpace(values["PRETTY_NAME"]); pretty != "" {
		return pretty
	}
	name := strings.TrimSpace(values["NAME"])
	version := strings.TrimSpace(values["VERSION_ID"])
	if name != "" && version != "" {
		return name + " " + version
	}
	return name
}

func parseOSRelease(input string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
		values[key] = value
	}
	return values
}
