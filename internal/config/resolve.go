package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Resolver handles environment variable resolution in config values.
type Resolver struct {
	env map[string]string
}

// NewResolver creates a Resolver using the current environment.
func NewResolver() *Resolver {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if idx := strings.Index(e, "="); idx != -1 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	return &Resolver{env: env}
}

// NewResolverWithEnv creates a Resolver with a custom environment map.
func NewResolverWithEnv(env map[string]string) *Resolver {
	return &Resolver{env: env}
}

// varPattern matches $VAR and ${VAR} patterns.
var varPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}|\$([a-zA-Z_][a-zA-Z0-9_]*)`)

// Resolve expands environment variables in a string.
// Supports $VAR and ${VAR} syntax.
// Returns an error if a referenced variable is not set.
func (r *Resolver) Resolve(value string) (string, error) {
	if !strings.Contains(value, "$") {
		return value, nil
	}

	var errs []string
	result := varPattern.ReplaceAllStringFunc(value, func(match string) string {
		var name string
		if strings.HasPrefix(match, "${") {
			name = match[2 : len(match)-1]
		} else {
			name = match[1:]
		}

		if val, ok := r.env[name]; ok {
			return val
		}
		errs = append(errs, name)
		return match
	})

	if len(errs) > 0 {
		return "", fmt.Errorf("undefined environment variables: %s", strings.Join(errs, ", "))
	}

	return result, nil
}

// MustResolve resolves a value or returns an empty string on error.
func (r *Resolver) MustResolve(value string) string {
	resolved, err := r.Resolve(value)
	if err != nil {
		return ""
	}
	return resolved
}
