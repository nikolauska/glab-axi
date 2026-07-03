package main

import (
	"strings"
	"testing"
)

func TestRenderListQuotesDelimiter(t *testing.T) {
	got := renderList("issues", []any{map[string]any{"iid": float64(1), "title": "one, two", "state": "opened"}}, []string{"iid", "title", "state"})
	want := "issues[1]{iid,title,state}:\n  1,\"one, two\",opened\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSelectFieldsTruncates(t *testing.T) {
	got, truncated := selectFields(map[string]any{"description": strings.Repeat("x", previewLimit+1)}, []string{"description"}, false)
	if !truncated || !strings.Contains(got["description"].(string), "truncated") {
		t.Fatal("expected truncation marker")
	}
}

func TestTakeRepoFlag(t *testing.T) {
	repo, rest, err := takeValue([]string{"issue", "list", "--repo=group/project"}, "-R", "--repo")
	if err != nil || repo != "group/project" || strings.Join(rest, " ") != "issue list" {
		t.Fatalf("%q %v %v", repo, rest, err)
	}
}

func TestEncodeNestedAPIArray(t *testing.T) {
	got := encodeTOON(map[string]any{"items": []any{map[string]any{"id": float64(1), "name": "first"}}})
	want := "items[1]:\n  - id: 1\n    name: first"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
