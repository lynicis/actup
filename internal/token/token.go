package token

import (
	"os"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func Resolve(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if envToken := os.Getenv("GITHUB_TOKEN"); envToken != "" {
		return envToken
	}

	cmd := execCommand("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}
