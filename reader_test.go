package env_test

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/17xande-dev/env"
)

const setting = "SETTING"

func lookup(values map[string]string) env.Lookup {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func assertSettingError(t *testing.T, err error, want string, values map[string]string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatal("unexpected configuration error")
		}
		return
	}
	if err == nil || !strings.HasPrefix(err.Error(), setting+": ") || !strings.Contains(err.Error(), want) {
		t.Fatalf("want a %s problem containing %q", setting, want)
	}
	if value := values[setting]; value != "" && strings.Contains(err.Error(), value) {
		t.Fatal("configuration error echoed the setting value")
	}
}

func TestString(t *testing.T) {
	for _, tt := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"unset", nil, "default"},
		{"empty", map[string]string{setting: ""}, "default"},
		{"as_given", map[string]string{setting: "  MiXeD \t"}, "  MiXeD \t"},
		{"only_space", map[string]string{setting: " "}, " "},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := env.New(lookup(tt.env))
			if got := r.String(setting, "default"); got != tt.want {
				t.Fatal("String did not return the expected setting")
			}
			assertSettingError(t, r.Err(), "", tt.env)
		})
	}
}

func TestRequired(t *testing.T) {
	for _, tt := range []struct {
		name    string
		env     map[string]string
		want    string
		problem string
	}{
		{"unset", nil, "", "is required"},
		{"empty", map[string]string{setting: ""}, "", "is required"},
		{"as_given", map[string]string{setting: "  MiXeD \t"}, "  MiXeD \t", ""},
		{"only_space", map[string]string{setting: " "}, " ", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := env.New(lookup(tt.env))
			if got := r.Required(setting); got != tt.want {
				t.Fatal("Required did not return the expected setting")
			}
			assertSettingError(t, r.Err(), tt.problem, tt.env)
		})
	}
}

func TestBool(t *testing.T) {
	const problem = `must be "true" or "false"`
	for _, tt := range []struct {
		name    string
		env     map[string]string
		def     bool
		want    bool
		problem string
	}{
		{"true", map[string]string{setting: "true"}, false, true, ""},
		{"false", map[string]string{setting: "false"}, true, false, ""},
		{"uppercase", map[string]string{setting: "TRUE"}, true, true, problem},
		{"titlecase", map[string]string{setting: "True"}, true, true, problem},
		{"typo", map[string]string{setting: "ture"}, true, true, problem},
		{"one", map[string]string{setting: "1"}, false, false, problem},
		{"zero", map[string]string{setting: "0"}, true, true, problem},
		{"yes", map[string]string{setting: "yes"}, true, true, problem},
		{"on", map[string]string{setting: "on"}, true, true, problem},
		{"leading_space", map[string]string{setting: " true"}, true, true, problem},
		{"trailing_space", map[string]string{setting: "true "}, true, true, problem},
		{"empty", map[string]string{setting: ""}, true, true, ""},
		{"unset", nil, true, true, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := env.New(lookup(tt.env))
			assertBool(t, r, tt.def, tt.want, tt.problem, tt.env)
		})
	}
}

func assertBool(t *testing.T, r *env.Reader, def, want bool, problem string, values map[string]string) {
	t.Helper()
	if got := r.Bool(setting, def); got != want {
		t.Errorf("Bool = %t, want %t", got, want)
	}
	assertSettingError(t, r.Err(), problem, values)
}

func TestInt(t *testing.T) {
	const syntax = "must be a decimal integer"
	for _, tt := range []struct {
		name    string
		env     map[string]string
		min     int
		max     int
		want    int
		problem string
	}{
		{"unset", nil, 1, 8, 3, ""},
		{"empty", map[string]string{setting: ""}, 1, 8, 3, ""},
		{"min", map[string]string{setting: "1"}, 1, 8, 1, ""},
		{"max", map[string]string{setting: "8"}, 1, 8, 8, ""},
		{"below_min", map[string]string{setting: "0"}, 1, 8, 3, "must be between 1 and 8 inclusive"},
		{"above_max", map[string]string{setting: "9"}, 1, 8, 3, "must be between 1 and 8 inclusive"},
		{"zero_max_inclusive", map[string]string{setting: "0"}, -8, 0, 0, ""},
		{"zero_max_is_a_bound", map[string]string{setting: "2"}, -8, 0, 3, "must be between -8 and 0 inclusive"},
		{"fraction", map[string]string{setting: "1.0"}, 1, 8, 3, syntax},
		{"hex", map[string]string{setting: "0x10"}, 1, 8, 3, syntax},
		{"space", map[string]string{setting: " 5"}, 1, 8, 3, syntax},
		{"plus_sign", map[string]string{setting: "+5"}, 1, 8, 5, ""},
		{"overflow", map[string]string{setting: "99999999999999999999999999"}, 1, 8, 3, syntax},
		{"misplaced_value", map[string]string{setting: "opaque:fixture"}, 1, 8, 3, syntax},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := env.New(lookup(tt.env))
			if got := r.Int(setting, 3, tt.min, tt.max); got != tt.want {
				t.Error("Int did not return the expected setting")
			}
			assertSettingError(t, r.Err(), tt.problem, tt.env)
		})
	}
}

func TestDuration(t *testing.T) {
	const (
		second = time.Second
		minute = time.Minute
		def    = 5 * second
		syntax = "must be a Go duration"
	)
	for _, tt := range []struct {
		name    string
		env     map[string]string
		min     time.Duration
		max     time.Duration
		want    time.Duration
		problem string
	}{
		{"unset", nil, second, 2 * minute, def, ""},
		{"empty", map[string]string{setting: ""}, second, 2 * minute, def, ""},
		{"seconds", map[string]string{setting: "90s"}, second, 2 * minute, 90 * second, ""},
		{"min", map[string]string{setting: "1s"}, second, 2 * minute, second, ""},
		{"max", map[string]string{setting: "2m"}, second, 2 * minute, 2 * minute, ""},
		{"below_min", map[string]string{setting: "0"}, second, 2 * minute, def, "must be at least 1s"},
		{"above_max", map[string]string{setting: "3m"}, second, 2 * minute, def, "must be at most 2m0s"},
		{"zero_max_is_unbounded", map[string]string{setting: "3m"}, second, 0, 3 * minute, ""},
		{"unbounded_max_still_checks_min", map[string]string{setting: "0"}, second, 0, def, "must be at least 1s"},
		{"zero_with_zero_min", map[string]string{setting: "0"}, 0, 0, 0, ""},
		{"syntax", map[string]string{setting: "soon"}, second, 2 * minute, def, syntax},
		{"space", map[string]string{setting: " 1h"}, second, 2 * time.Hour, def, syntax},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := env.New(lookup(tt.env))
			if got := r.Duration(setting, def, tt.min, tt.max); got != tt.want {
				t.Error("Duration did not return the expected setting")
			}
			assertSettingError(t, r.Err(), tt.problem, tt.env)
		})
	}
}

func TestList(t *testing.T) {
	def := []string{"first", " second "}
	for _, tt := range []struct {
		name string
		env  map[string]string
		want []string
	}{
		{"unset", nil, def},
		{"empty", map[string]string{setting: ""}, def},
		{"split_and_trim", map[string]string{setting: "a, b,,c"}, []string{"a", "b", "c"}},
		{"preserve_order_and_duplicates", map[string]string{setting: "\tb, a ,b, A "}, []string{"b", "a", "b", "A"}},
		{"only_empty_items", map[string]string{setting: " ,\t, "}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := env.New(lookup(tt.env))
			if got := r.List(setting, def); !slices.Equal(got, tt.want) {
				t.Fatal("List did not return the expected items")
			}
			assertSettingError(t, r.Err(), "", tt.env)
		})
	}
}

func TestOneOf(t *testing.T) {
	const problem = "must exactly match an allowed value"
	for _, tt := range []struct {
		name    string
		env     map[string]string
		want    string
		problem string
	}{
		{"unset", nil, "info", ""},
		{"empty", map[string]string{setting: ""}, "info", ""},
		{"exact", map[string]string{setting: "debug"}, "debug", ""},
		{"case", map[string]string{setting: "Debug"}, "info", problem},
		{"space", map[string]string{setting: " debug "}, "info", problem},
		{"unknown", map[string]string{setting: "opaque"}, "info", problem},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := env.New(lookup(tt.env))
			if got := r.OneOf(setting, "info", "debug", "info"); got != tt.want {
				t.Error("OneOf did not return the expected setting")
			}
			assertSettingError(t, r.Err(), tt.problem, tt.env)
		})
	}
}

func TestReaderCollectsProblemsInReadOrder(t *testing.T) {
	r := env.New(lookup(map[string]string{"Z_FLAG": "ture", "A_COUNT": "opaque"}))
	if got := r.Bool("Z_FLAG", true); !got {
		t.Error("Bool did not return its default")
	}
	if got := r.Int("A_COUNT", 5, 1, 9); got != 5 {
		t.Error("Int did not return its default")
	}
	if got := r.Required("M_LABEL"); got != "" {
		t.Error("Required did not return an empty string")
	}
	const want = "Z_FLAG: must be \"true\" or \"false\"\nA_COUNT: must be a decimal integer\nM_LABEL: is required"
	if err := r.Err(); err == nil || err.Error() != want {
		t.Fatalf("want all problems in read order:\n%s", want)
	}
}

func TestFail(t *testing.T) {
	r := env.New(lookup(nil))
	if r.Err() != nil {
		t.Fatal("fresh Reader should have no problems")
	}
	r.Fail("BASE_URL", "x")
	if err := r.Err(); err == nil || err.Error() != "BASE_URL: x" {
		t.Fatal("Fail should record the key and problem")
	}
}

func TestLookupEnv(t *testing.T) {
	const key = "ENV_READER_TEST_SETTING"
	r := env.New(os.LookupEnv)
	t.Setenv(key, "original")
	if got := r.String(key, "default"); got != "original" {
		t.Fatal("Reader did not read the environment")
	}
	t.Setenv(key, "")
	if got := r.String(key, "default"); got != "default" {
		t.Fatal("empty environment setting did not select the default")
	}
}
