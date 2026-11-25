package env

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Lookup looks up a setting, as os.LookupEnv does.
type Lookup func(key string) (value string, ok bool)

// Reader reads settings and collects every problem. It is not safe for concurrent use.
type Reader struct {
	lookup   Lookup
	problems []error
}

// New returns a reader that reads settings through lookup.
func New(lookup Lookup) *Reader {
	return &Reader{lookup: lookup}
}

// String returns the setting as given, or def if it is unset or empty.
func (r *Reader) String(key, def string) string {
	return r.text(key, def)
}

// Required returns the setting as given, or records a problem and returns an
// empty string if it is unset or empty.
func (r *Reader) Required(key string) string {
	value := r.text(key, "")
	if value == "" {
		r.Fail(key, "is required")
	}
	return value
}

// Bool reads exactly "true" or "false".
// It returns def if the setting is unset, empty, or invalid.
func (r *Reader) Bool(key string, def bool) bool {
	switch r.text(key, "") {
	case "":
		return def
	case "true":
		return true
	case "false":
		return false
	default:
		r.Fail(key, `must be "true" or "false"`)
		return def
	}
}

// Int reads a decimal integer between min and max, inclusive.
// It returns def if the setting is unset, empty, or invalid.
func (r *Reader) Int(key string, def, min, max int) int {
	value := r.text(key, "")
	if value == "" {
		return def
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		r.Fail(key, "must be a decimal integer")
		return def
	}
	if n < min || n > max {
		r.Fail(key, fmt.Sprintf("must be between %d and %d inclusive", min, max))
		return def
	}
	return n
}

// Duration reads a duration accepted by [time.ParseDuration] between min and
// max, inclusive. A max of zero means no upper bound.
// It returns def if the setting is unset, empty, or invalid.
func (r *Reader) Duration(key string, def, min, max time.Duration) time.Duration {
	value := r.text(key, "")
	if value == "" {
		return def
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		r.Fail(key, "must be a Go duration")
		return def
	}
	if d < min {
		r.Fail(key, "must be at least "+min.String())
		return def
	}
	if max != 0 && d > max {
		r.Fail(key, "must be at most "+max.String())
		return def
	}
	return d
}

// List splits the setting on commas, trims spaces from each item, and drops
// empty items. It returns def if the setting itself is unset or empty.
func (r *Reader) List(key string, def []string) []string {
	value := r.text(key, "")
	if value == "" {
		return def
	}
	var items []string
	for part := range strings.SplitSeq(value, ",") {
		if item := strings.TrimSpace(part); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// OneOf reads a setting that exactly matches one of the allowed values.
// It returns def if the setting is unset, empty, or invalid.
func (r *Reader) OneOf(key, def string, allowed ...string) string {
	value := r.text(key, "")
	if value == "" {
		return def
	}
	if !slices.Contains(allowed, value) {
		r.Fail(key, "must exactly match an allowed value")
		return def
	}
	return value
}

// Fail records a problem with key. Use a fixed description of what is wanted;
// problem must not include input values or raw parser errors.
func (r *Reader) Fail(key, problem string) {
	r.problems = append(r.problems, errors.New(key+": "+problem))
}

// Err returns the problems joined in the order they were recorded, or nil if
// no problems have been recorded.
func (r *Reader) Err() error {
	return errors.Join(r.problems...)
}

func (r *Reader) text(key, def string) string {
	value, ok := r.lookup(key)
	if !ok || value == "" {
		return def
	}
	return value
}
