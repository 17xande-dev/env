// Package envtest provides helpers for checking configuration documentation
// against the environment keys read by tests.
package envtest

import (
	"bufio"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Recorder reads from a map and tracks every lookup, including misses.
// It is not safe for concurrent use.
type Recorder struct {
	values map[string]string
	keys   map[string]struct{}
}

// NewRecorder constructs a recorder for values, which may be nil.
func NewRecorder(values map[string]string) *Recorder {
	return &Recorder{values: values, keys: make(map[string]struct{})}
}

// Lookup records key and returns its value and presence, like os.LookupEnv.
func (r *Recorder) Lookup(key string) (string, bool) {
	if r.keys == nil {
		r.keys = make(map[string]struct{})
	}
	r.keys[key] = struct{}{}
	value, ok := r.values[key]
	return value, ok
}

// Keys returns the sorted, unique keys read so far.
func (r *Recorder) Keys() []string { return slices.Sorted(maps.Keys(r.keys)) }

var (
	assignment = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=`)
	keyCell    = regexp.MustCompile("^`([A-Za-z_][A-Za-z0-9_]*)`$")
)

// EnvExample returns the keys of KEY= and # KEY= lines, in file order.
// It fails the test if a key appears twice or if no keys are found.
func EnvExample(t testing.TB, path string) []string {
	t.Helper()
	var keys []string
	seen := make(map[string]bool)
	scanLines(t, path, func(line string) bool {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))
		if match := assignment.FindStringSubmatch(line); match != nil {
			key := match[1]
			if seen[key] {
				t.Fatalf("duplicate environment key %s", key)
			}
			seen[key] = true
			keys = append(keys, key)
		}
		return true
	})
	if len(keys) == 0 {
		t.Fatal("no environment keys in example")
	}
	return keys
}

// MarkdownTable returns the backticked keys in the first column under heading
// (including its Markdown heading marks), in file order. It stops at the next
// heading of the same or higher level and fails if the section has no keys.
func MarkdownTable(t testing.TB, path, heading string) []string {
	t.Helper()
	heading = strings.TrimSpace(heading)
	level := headingLevel(heading)
	if level == 0 {
		t.Fatalf("invalid Markdown heading %q", heading)
	}
	var keys []string
	found := false
	scanLines(t, path, func(line string) bool {
		line = strings.TrimSpace(line)
		if !found {
			found = line == heading
			return true
		}
		if next := headingLevel(line); next > 0 && next <= level {
			return false
		}
		cell, _, ok := strings.Cut(strings.TrimPrefix(line, "|"), "|")
		if ok {
			if match := keyCell.FindStringSubmatch(strings.TrimSpace(cell)); match != nil {
				keys = append(keys, match[1])
			}
		}
		return true
	})
	if !found {
		t.Fatalf("missing Markdown heading %q", heading)
	}
	if len(keys) == 0 {
		t.Fatalf("no environment keys under Markdown heading %q", heading)
	}
	return keys
}

func headingLevel(line string) int {
	marks, _, ok := strings.Cut(line, " ")
	if ok && len(marks) <= 6 && strings.Trim(marks, "#") == "" {
		return len(marks)
	}
	return 0
}

// Documented fails the test if no keys were read or if the read and
// documented key sets differ. Read every key unconditionally and validate
// conditionally so that the recorder can see every setting.
func Documented(t testing.TB, what string, read, documented []string) {
	t.Helper()
	if len(read) == 0 {
		t.Errorf("%s: no environment keys were read", what)
	}
	compareKeys(t, what, read, documented, "read but not documented")
	compareKeys(t, what, documented, read, "documented but not read")
}

func compareKeys(t testing.TB, what string, keys, other []string, problem string) {
	t.Helper()
	sorted := slices.Compact(slices.Sorted(slices.Values(keys)))
	for _, key := range sorted {
		if !slices.Contains(other, key) {
			t.Errorf("%s: %s: %s", what, key, problem)
		}
	}
}

func scanLines(t testing.TB, path string, visit func(string) bool) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal("cannot open documentation file")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() && visit(scanner.Text()) {
	}
	if scanner.Err() != nil {
		t.Fatal("cannot read documentation file")
	}
}
