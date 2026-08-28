package languages

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// The whole point of a string id: a language the client has never heard of must
// still be usable, so the server can add one without a client release.
func TestGetToleratesUnknownLanguage(t *testing.T) {
	got := Get("brainfuck")

	if got.ID != "brainfuck" {
		t.Errorf("ID = %q, want %q", got.ID, "brainfuck")
	}
	if got.DisplayName == "" {
		t.Error("DisplayName is empty; an unknown language should still be presentable")
	}
	if got.CommentPrefix == "" {
		t.Error("CommentPrefix is empty; FallbackStub would produce a bare string")
	}
}

func TestGetKnownLanguage(t *testing.T) {
	got := Get("python")

	if got.DisplayName != "Python" {
		t.Errorf("DisplayName = %q", got.DisplayName)
	}
	if got.CommentPrefix != "#" {
		t.Errorf("CommentPrefix = %q, want #", got.CommentPrefix)
	}
}

// A fallback buffer must not be a syntax error the moment it opens.
func TestFallbackStubUsesTheRightComment(t *testing.T) {
	tests := []struct {
		id     string
		prefix string
	}{
		{"python", "#"},
		{"ruby", "#"},
		{"go", "//"},
		{"cpp", "//"},
		{"kotlin", "//"},
		{"nobody-has-heard-of-this", "//"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := Get(tt.id).FallbackStub()
			if !strings.HasPrefix(got, tt.prefix+" ") {
				t.Errorf("FallbackStub() = %q, want it to start with %q", got, tt.prefix)
			}
		})
	}
}

func TestStaticSupported(t *testing.T) {
	t.Run("from config", func(t *testing.T) {
		got, err := NewStatic([]string{"kotlin", "rust"}).Supported(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"kotlin", "rust"}; !reflect.DeepEqual(IDs(got), want) {
			t.Errorf("got %v, want %v", IDs(got), want)
		}
	})

	t.Run("empty config falls back to defaults", func(t *testing.T) {
		got, err := NewStatic(nil).Supported(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(IDs(got), DefaultIDs()) {
			t.Errorf("got %v, want %v", IDs(got), DefaultIDs())
		}
	})
}

func TestUnion(t *testing.T) {
	supported := []Language{Get("python"), Get("go")}

	tests := []struct {
		name string
		ids  []string
		want []string
	}{
		{"adds unknown", []string{"kotlin"}, []string{"python", "go", "kotlin"}},
		{"skips duplicates", []string{"go", "python"}, []string{"python", "go"}},
		{"preserves supported order", []string{"rust", "go"}, []string{"python", "go", "rust"}},
		{"no extras", nil, []string{"python", "go"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IDs(Union(supported, tt.ids)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
