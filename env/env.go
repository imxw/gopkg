// Package env provides environment-variable configuration loading:
// struct parsing (re-exported from github.com/caarlos0/env/v11 — the
// maintained zero-dependency community standard) plus small typed getters
// with defaults for non-struct call sites.
package env

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Parse parses environment variables into the struct fields of v, driven
// by `env:"KEY"` tags. See caarlos0/env docs for tag options (default,
// required, separator, …).
func Parse(v interface{}) error {
	return env.Parse(v)
}

// ParseAs is the generic form of Parse.
func ParseAs[T any]() (T, error) {
	return env.ParseAs[T]()
}

// Get returns the raw value of an environment variable ("" if unset).
func Get(key string) string {
	return os.Getenv(key)
}

// Has reports whether the variable is set (even to an empty string).
func Has(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

// GetDefault returns the value of key, or def if unset OR empty — the
// caller-owned-fallback shape (strconv family helpers share the contract:
// invalid/absent both fall back).
func GetDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// GetInt returns the int value of key, or def if unset/invalid.
func GetInt(key string, def int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

// GetInt64 returns the int64 value of key, or def if unset/invalid.
func GetInt64(key string, def int64) int64 {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return def
	}
	return n
}

// GetBool returns the bool value of key (strconv.ParseBool semantics:
// 1/t/T/TRUE/true/True), or def if unset/invalid.
func GetBool(key string, def bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}
	return b
}

// GetDuration returns the time.Duration value of key (time.ParseDuration
// semantics: "10s", "2m"), or def if unset/invalid.
func GetDuration(key string, def time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return def
	}
	return d
}

// ParseEmptyAsUnset parses environment variables into v like Parse, with
// one contract adjustment matching the Get/GetDefault family: an env var
// SET to the empty string counts as UNSET (falls to envDefault). Deployments
// commonly write `export FOO=` to "clear" a var — without this shim that
// would feed "" into typed fields (int → boot error, string → clobbers
// default). Invalid non-empty values still fail the parse (fail-fast),
// which is the intended contrast with the silent-fallback getters.
func ParseEmptyAsUnset(v interface{}) error {
	environ := os.Environ()
	m := make(map[string]string, len(environ))
	for _, kv := range environ {
		k, val, _ := strings.Cut(kv, "=")
		if val == "" {
			continue
		}
		m[k] = val
	}
	return env.ParseWithOptions(v, env.Options{Environment: m})
}

// Must returns the value of key; panics when unset or empty — for
// startup-required secrets that cannot have a safe default (fail fast at
// boot instead of first use).
func Must(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("env: required variable " + key + " is not set")
	}
	return v
}
