package envtest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/17xande-dev/env"
	"github.com/17xande-dev/env/envtest"
)

func TestRecorder(t *testing.T) {
	r := envtest.NewRecorder(map[string]string{"TOKEN": "fixture", "EMPTY": "", "UNREAD": "fixture"})
	if got, ok := r.Lookup("EMPTY"); got != "" || !ok {
		t.Error("present empty value should be recorded as present")
	}
	if got, ok := r.Lookup("MISSING"); got != "" || ok {
		t.Error("absent value should be recorded as absent")
	}
	reader := env.New(r.Lookup)
	for range 2 {
		if !reader.Secret("TOKEN").IsSet() || reader.Err() != nil {
			t.Fatal("reader did not find the direct credential")
		}
	}
	if got, want := r.Keys(), []string{"EMPTY", "MISSING", "TOKEN", "TOKEN_FILE"}; !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

func TestEnvExample(t *testing.T) {
	path := fixture(t, "  # comment\nFIRST=value\n# SECOND=\n  # THIRD=sample\nnot an assignment\n")
	if got, want := envtest.EnvExample(t, path), []string{"FIRST", "SECOND", "THIRD"}; !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

func TestEnvExampleRefusals(t *testing.T) {
	for _, tt := range []struct {
		name, contents, message string
	}{
		{"duplicate", "KEY=one\n# KEY=two\n", "duplicate environment key KEY"},
		{"empty", "# no settings yet\n", "no environment keys in example"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := fixture(t, tt.contents)
			checkFailure(t, func(tb testing.TB) { envtest.EnvExample(tb, path) }, tt.message)
		})
	}
}

func TestMarkdownTable(t *testing.T) {
	path := fixture(t, "# Intro\n| `IGNORE` | no |\n## Settings\n| Key | Description |\n| --- | --- |\n"+
		"| `FIRST` | first |\n| not_a_key | not backticked |\n| also_not_a_key | `IGNORE` |\n"+
		"### Details\n`SECOND` | second\n## Other\n| `IGNORE` | another section |\n")
	if got, want := envtest.MarkdownTable(t, path, "## Settings"), []string{"FIRST", "SECOND"}; !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

func TestMarkdownTableRefusals(t *testing.T) {
	for _, tt := range []struct {
		name, contents, message string
	}{
		{"missing_heading", "## Other\n| `KEY` | ignored |\n", `missing Markdown heading "## Settings"`},
		{"empty_section", "## Settings\n## Other\n| `KEY` | ignored |\n", `no environment keys under Markdown heading "## Settings"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := fixture(t, tt.contents)
			checkFailure(t, func(tb testing.TB) { envtest.MarkdownTable(tb, path, "## Settings") }, tt.message)
		})
	}
}

func TestDocumented(t *testing.T) {
	envtest.Documented(t, "settings", []string{"FIRST", "SECOND"}, []string{"SECOND", "FIRST"})
}

func TestDocumentedRefusals(t *testing.T) {
	t.Run("read_not_documented", func(t *testing.T) {
		path := fixture(t, "FIRST=\n")
		recorder := envtest.NewRecorder(nil)
		r := env.New(recorder.Lookup)
		r.String("FIRST", "")
		r.String("UNLISTED", "")
		checkFailure(t, func(tb testing.TB) {
			envtest.Documented(tb, ".env.example", recorder.Keys(), envtest.EnvExample(tb, path))
		}, ".env.example: UNLISTED: read but not documented")
	})
	t.Run("documented_not_read", func(t *testing.T) {
		path := fixture(t, "FIRST=\n# UNREAD=\n")
		recorder := envtest.NewRecorder(nil)
		env.New(recorder.Lookup).String("FIRST", "")
		checkFailure(t, func(tb testing.TB) {
			envtest.Documented(tb, ".env.example", recorder.Keys(), envtest.EnvExample(tb, path))
		}, ".env.example: UNREAD: documented but not read")
	})
	t.Run("readme_mismatch", func(t *testing.T) {
		path := fixture(t, "## Configuration\n| Key | Description |\n| --- | --- |\n| `TYPO` | value |\n")
		recorder := envtest.NewRecorder(nil)
		env.New(recorder.Lookup).String("SETTING", "")
		checkFailure(t, func(tb testing.TB) {
			envtest.Documented(tb, "README", recorder.Keys(), envtest.MarkdownTable(tb, path, "## Configuration"))
		}, "README: SETTING: read but not documented", "README: TYPO: documented but not read")
	})
	t.Run("conditional_read", func(t *testing.T) {
		path := fixture(t, "ENABLED=false\n# LIMIT=\n")
		recorder := envtest.NewRecorder(nil)
		r := env.New(recorder.Lookup)
		if r.Bool("ENABLED", false) {
			r.Int("LIMIT", 1, 1, 10)
		}
		checkFailure(t, func(tb testing.TB) {
			envtest.Documented(tb, ".env.example", recorder.Keys(), envtest.EnvExample(tb, path))
		}, ".env.example: LIMIT: documented but not read")
	})
	t.Run("empty_read", func(t *testing.T) {
		checkFailure(t, func(tb testing.TB) {
			envtest.Documented(tb, "settings", nil, nil)
		}, "settings: no environment keys were read")
	})
}

func fixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal("cannot create documentation fixture")
	}
	return path
}

type failRecorder struct {
	testing.TB
	messages []string
}

func (*failRecorder) Helper() {}

func (r *failRecorder) Errorf(format string, args ...any) {
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}

func (r *failRecorder) Fatal(args ...any) {
	r.messages = append(r.messages, fmt.Sprint(args...))
	runtime.Goexit()
}

func (r *failRecorder) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
	runtime.Goexit()
}

func checkFailure(t *testing.T, check func(testing.TB), want ...string) {
	t.Helper()
	done := make(chan []string, 1)
	go func() {
		r := &failRecorder{TB: t}
		defer func() { done <- r.messages }()
		check(r)
	}()
	if got := <-done; !slices.Equal(got, want) {
		t.Errorf("failure messages = %q, want %q", got, want)
	}
}
