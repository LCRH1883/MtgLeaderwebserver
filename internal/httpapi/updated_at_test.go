package httpapi

import (
	"testing"
	"time"
)

func TestParseUpdatedAtValid(t *testing.T) {
	got, err := parseUpdatedAt("2024-06-01T12:34:56.789Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2024, 6, 1, 12, 34, 56, 789000000, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("unexpected time: %s", got)
	}
}

func TestParseUpdatedAtInvalid(t *testing.T) {
	cases := []string{
		"2024-06-01T12:34:56Z",
		"2024-06-01T12:34:56.789+01:00",
		"not-a-time",
	}
	for _, tc := range cases {
		if _, err := parseUpdatedAt(tc); err == nil {
			t.Fatalf("expected error for %q", tc)
		}
	}
}
