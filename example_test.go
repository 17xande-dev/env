package env_test

import (
	"fmt"
	"time"

	"github.com/17xande-dev/env"
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
