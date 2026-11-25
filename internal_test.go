package env

import (
	"slices"
	"testing"
)

func TestTextIgnoresValueWhenLookupReportsUnset(t *testing.T) {
	var keys []string
	r := New(func(key string) (string, bool) {
		keys = append(keys, key)
		return "ignored", false
	})
	if got := r.text("FIRST", "default"); got != "default" {
		t.Error("text did not use the default for an unset key")
	}
	r.text("SECOND", "")
	r.text("FIRST", "")
	if want := []string{"FIRST", "SECOND", "FIRST"}; !slices.Equal(keys, want) {
		t.Error("text must look up each key on every read")
	}
}

func TestErrUnwrapsOrderedProblems(t *testing.T) {
	r := New(nil)
	r.Fail("SECOND", "first problem")
	r.Fail("FIRST", "second problem")
	r.Fail("SECOND", "another problem")

	joined, ok := r.Err().(interface{ Unwrap() []error })
	if !ok {
		t.Fatal("Err should join problems, not flatten them")
	}
	if got := joined.Unwrap(); !slices.Equal(got, r.problems) {
		t.Fatal("Err should preserve each recorded problem in order")
	}
}
