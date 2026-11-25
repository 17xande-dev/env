package env_test

import (
	"bytes"
	"encoding"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"testing"

	"github.com/17xande-dev/env"
)

const (
	credential = "fixture-credential"
	redacted   = "[redacted]"
	fileKey    = "TOKEN_FILE"
)

func TestSecretValue(t *testing.T) {
	for _, tt := range []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"unset", nil, ""},
		{"empty", map[string]string{"TOKEN": "", fileKey: ""}, ""},
		{"direct", map[string]string{"TOKEN": credential}, credential},
		{"direct_unchanged", map[string]string{"TOKEN": " \n" + credential + "\r\n", fileKey: ""}, " \n" + credential + "\r\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := env.New(lookup(tt.values))
			assertSecretValue(t, r.Secret("TOKEN"), tt.want)
			if r.Err() != nil {
				t.Fatal("unexpected configuration error")
			}
		})
	}
}

func assertSecretValue(t *testing.T, got env.Secret, want string) {
	t.Helper()
	if got.Reveal() != want || got.IsSet() != (want != "") {
		t.Fatal("unexpected secret value or set state")
	}
}

func TestSecretFormatting(t *testing.T) {
	secret := env.New(lookup(map[string]string{"TOKEN": credential})).Secret("TOKEN")
	var unset env.Secret
	for _, tt := range []struct{ name, verb string }{
		{"v", "%v"}, {"plus_v", "%+v"}, {"sharp_v", "%#v"},
		{"string", "%s"}, {"quoted", "%q"}, {"hex", "%x"},
		{"decimal", "%d"}, {"upper_hex", "%X"}, {"unicode", "%U"}, {"bool", "%t"},
		{"width_precision", "%8.3v"}, {"left_aligned", "%-20s"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.verb == "%#v" {
				// Format takes precedence over GoString, so check both contracts.
				for _, tt := range []struct {
					value any
					want  string
				}{{secret, redacted}, {unset, ""}} {
					s, ok := tt.value.(fmt.GoStringer)
					if !ok || s.GoString() != tt.want {
						t.Fatal("GoString did not provide the expected display value")
					}
				}
			}
			if fmt.Sprintf(tt.verb, secret) != redacted || fmt.Sprintf(tt.verb, unset) != "" {
				t.Fatal("formatting did not use the expected display value")
			}
		})
	}
	t.Run("type_and_pointer", func(t *testing.T) {
		// fmt handles %T and %p itself, without calling Format.
		if got := fmt.Sprintf("%T", secret); got != "env.Secret" {
			t.Fatal("unexpected formatted type")
		}
		if got := fmt.Sprintf("%p", &secret); !strings.HasPrefix(got, "0x") || strings.Contains(got, credential) {
			t.Fatal("pointer formatting did not show only an address")
		}
	})
	for _, tt := range []struct {
		name  string
		value any
		want  string
	}{
		{"pointer", &secret, redacted},
		{"map", map[string]env.Secret{"token": secret}, "map[token:" + redacted + "]"},
		{"slice", []env.Secret{secret}, "[" + redacted + "]"},
		{"interface_field", struct{ Token any }{secret}, "{" + redacted + "}"},
		{"nested_struct", struct{ Inner struct{ Token env.Secret } }{struct{ Token env.Secret }{secret}}, "{{" + redacted + "}}"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if fmt.Sprint(tt.value) != tt.want || fmt.Sprintln(tt.value) != tt.want+"\n" {
				t.Fatal("formatted value did not use the expected display value")
			}
		})
	}
	t.Run("struct", func(t *testing.T) {
		cfg := struct{ Token env.Secret }{secret}
		if fmt.Sprint(cfg) != "{"+redacted+"}" || fmt.Sprintf("%+v", cfg) != "{Token:"+redacted+"}" {
			t.Fatal("formatted struct did not redact its secret")
		}
	})
	t.Run("unexported_field", func(t *testing.T) {
		cfg := struct{ token env.Secret }{secret}
		for _, format := range []string{"%+v", "%#v", "%x"} {
			out := fmt.Sprintf(format, cfg)
			if strings.Contains(out, credential) || strings.Contains(out, fmt.Sprintf("%x", credential)) {
				t.Fatal("formatted struct disclosed an unexported secret")
			}
		}
	})
	t.Run("error", func(t *testing.T) {
		if fmt.Errorf("invalid credential: %v", secret).Error() != "invalid credential: "+redacted {
			t.Fatal("formatted error did not redact its secret")
		}
	})
	t.Run("log", func(t *testing.T) {
		var out bytes.Buffer
		log.New(&out, "", 0).Printf("token=%d", secret)
		if out.String() != "token="+redacted+"\n" {
			t.Fatal("log did not redact its secret")
		}
	})
}

func TestSecretSlog(t *testing.T) {
	secret := env.New(lookup(map[string]string{"TOKEN": credential})).Secret("TOKEN")
	for _, mode := range []string{"json", "text"} {
		t.Run(mode, func(t *testing.T) {
			var out bytes.Buffer
			var handler slog.Handler
			switch mode {
			case "json":
				handler = slog.NewJSONHandler(&out, nil)
			case "text":
				handler = slog.NewTextHandler(&out, nil)
			}
			for _, tt := range []struct {
				secret env.Secret
				want   string
			}{{secret, redacted}, {env.Secret{}, ""}} {
				out.Reset()
				slog.New(handler).WithGroup("settings").Info("configured", "token", tt.secret)
				if mode == "json" {
					// The JSON handler can fall back to MarshalJSON if LogValue is missing.
					s, ok := any(tt.secret).(slog.LogValuer)
					if !ok || !s.LogValue().Equal(slog.StringValue(tt.want)) {
						t.Fatal("LogValue did not provide the expected display value")
					}
					var record struct {
						Settings map[string]string
					}
					if err := json.Unmarshal(out.Bytes(), &record); err != nil || record.Settings["token"] != tt.want {
						t.Fatal("JSON handler did not emit the expected display value")
					}
				} else {
					want := tt.want
					if want == "" {
						want = `""`
					}
					if !strings.HasSuffix(out.String(), " settings.token="+want+"\n") {
						t.Fatal("text handler did not emit the expected display value")
					}
				}
			}
		})
	}
}

func TestSecretXML(t *testing.T) {
	secret := env.New(lookup(map[string]string{"TOKEN": credential})).Secret("TOKEN")
	type settings struct {
		XMLName xml.Name   `xml:"settings"`
		Token   env.Secret `xml:"token"`
		Attr    env.Secret `xml:"token,attr"`
	}
	for _, tt := range []struct {
		secret env.Secret
		want   string
	}{
		{secret, `<settings token="[redacted]"><token>[redacted]</token></settings>`},
		{env.Secret{}, `<settings token=""><token></token></settings>`},
	} {
		got, err := xml.Marshal(settings{Token: tt.secret, Attr: tt.secret})
		if err != nil || string(got) != tt.want {
			t.Fatal("XML did not encode the expected display value")
		}
	}
}

func TestSecretJSON(t *testing.T) {
	secret := env.New(lookup(map[string]string{"TOKEN": credential})).Secret("TOKEN")
	for _, tt := range []struct {
		secret env.Secret
		want   string
	}{{secret, `{"token":"[redacted]"}`}, {env.Secret{}, `{"token":""}`}} {
		got, err := json.Marshal(struct {
			Token env.Secret `json:"token"`
		}{tt.secret})
		if err != nil || string(got) != tt.want {
			t.Fatal("JSON struct did not encode the expected display value")
		}
	}
}

func TestSecretDisplayMethods(t *testing.T) {
	secret := env.New(lookup(map[string]string{"TOKEN": credential})).Secret("TOKEN")
	// Consumer fallbacks may also redact, so assert each contract independently.
	for _, tt := range []struct {
		value any
		want  string
	}{{secret, redacted}, {env.Secret{}, ""}} {
		t.Run("String", func(t *testing.T) {
			s, ok := tt.value.(fmt.Stringer)
			if !ok || s.String() != tt.want {
				t.Fatal("String did not provide the expected display value")
			}
		})
		t.Run("MarshalJSON", func(t *testing.T) {
			s, ok := tt.value.(json.Marshaler)
			if !ok {
				t.Fatal("Secret must implement json.Marshaler")
			}
			result, err := s.MarshalJSON()
			if err != nil || string(result) != `"`+tt.want+`"` {
				t.Fatal("MarshalJSON did not provide the expected display value")
			}
		})
		t.Run("MarshalText", func(t *testing.T) {
			s, ok := tt.value.(encoding.TextMarshaler)
			if !ok {
				t.Fatal("Secret must implement encoding.TextMarshaler")
			}
			result, err := s.MarshalText()
			if err != nil || string(result) != tt.want {
				t.Fatal("MarshalText did not provide the expected display value")
			}
		})
	}
}
