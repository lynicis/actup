package token

import (
	"os"
	"os/exec"
	"strings"
)

type commandRunner func(name string, arg ...string) *exec.Cmd

type Resolver struct {
	run commandRunner
}

func NewResolver() *Resolver {
	return &Resolver{
		run: exec.Command,
	}
}

func (r *Resolver) Resolve(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if envToken := os.Getenv("GITHUB_TOKEN"); envToken != "" {
		return envToken
	}

	cmd := r.run("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}
