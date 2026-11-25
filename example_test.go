package env_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/17xande-dev/env"
	"github.com/17xande-dev/env/envtest"
)

func ExampleReader() {
	values := map[string]string{
		"LABEL":   "worker",
		"QUEUE":   "incoming",
		"ACTIVE":  "true",
		"LIMIT":   "12",
		"TIMEOUT": "90s",
		"TAGS":    "small, fast,,batch",
		"MODE":    "brief",
	}
	r := env.New(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	label := r.String("LABEL", "untitled")
	queue := r.Required("QUEUE")
	active := r.Bool("ACTIVE", false)
	limit := r.Int("LIMIT", 10, 1, 100)
	timeout := r.Duration("TIMEOUT", time.Minute, time.Second, 0)
	tags := r.List("TAGS", nil)
	mode := r.OneOf("MODE", "full", "brief", "full")
	if err := r.Err(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(label, queue, active, limit, timeout, tags, mode)
	// Output: worker incoming true 12 1m30s [small fast batch] brief
}

func ExampleReader_Err() {
	r := env.New(func(key string) (string, bool) {
		if key == "ACTIVE" {
			return "ture", true
		}
		return "", false
	})
	r.Bool("ACTIVE", true)
	r.Required("QUEUE")
	r.Fail("MODE", "requires at least one target")
	fmt.Println(r.Err())
	// Output:
	// ACTIVE: must be "true" or "false"
	// QUEUE: is required
	// MODE: requires at least one target
}

func ExampleReader_Secret() {
	values := map[string]string{"PUBLISH": "true", "PUBLISH_TOKEN": "fixture"}
	cfg, err := load(envtest.NewRecorder(values).Lookup)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(cfg.Publish, cfg.Token.IsSet(), cfg.Token)
	// Output: true true [redacted]
}

type Config struct {
	Label   string
	Publish bool
	Token   env.Secret
}

func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup env.Lookup) (Config, error) {
	r := env.New(lookup)
	cfg := Config{
		Label:   r.String("LABEL", "worker"),
		Publish: r.Bool("PUBLISH", false),
		Token:   r.Secret("PUBLISH_TOKEN"),
	}
	if cfg.Publish && !cfg.Token.IsSet() {
		r.Fail("PUBLISH_TOKEN", "is required when publishing is enabled (or set PUBLISH_TOKEN_FILE)")
	}
	if err := r.Err(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func TestConfigDocumentation(t *testing.T) {
	recorder := envtest.NewRecorder(nil)
	if _, err := load(recorder.Lookup); err != nil {
		t.Fatal(err)
	}
	read := recorder.Keys()
	envtest.Documented(t, ".env.example", read, envtest.EnvExample(t, "testdata/.env.example"))
	envtest.Documented(t, "README", read, envtest.MarkdownTable(t, "README.md", "## Configuration"))
}
