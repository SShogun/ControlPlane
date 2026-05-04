package validator

import "testing"

func TestNotBlank(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"non-empty string", "hello", true},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"tab only", "\t", false},
		{"value with surrounding spaces", "  hi  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NotBlank(tt.value); got != tt.want {
				t.Errorf("NotBlank(%q) = %v; want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestMaxChars(t *testing.T) {
	tests := []struct {
		name  string
		value string
		max   int
		want  bool
	}{
		{"under limit", "hello", 10, true},
		{"at limit", "hello", 5, true},
		{"over limit", "hello world", 5, false},
		{"empty string", "", 5, true},
		{"unicode chars", "héllo", 5, true},
		{"unicode over limit", "héllo!", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxChars(tt.value, tt.max); got != tt.want {
				t.Errorf("MaxChars(%q, %d) = %v; want %v", tt.value, tt.max, got, tt.want)
			}
		})
	}
}

func TestMinChars(t *testing.T) {
	tests := []struct {
		name  string
		value string
		min   int
		want  bool
	}{
		{"over minimum", "hello", 3, true},
		{"at minimum", "hi", 2, true},
		{"under minimum", "h", 2, false},
		{"empty string", "", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MinChars(tt.value, tt.min); got != tt.want {
				t.Errorf("MinChars(%q, %d) = %v; want %v", tt.value, tt.min, got, tt.want)
			}
		})
	}
}

func TestPositiveInt(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  bool
	}{
		{"positive", 1, true},
		{"large positive", 9999, true},
		{"zero", 0, false},
		{"negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PositiveInt(tt.value); got != tt.want {
				t.Errorf("PositiveInt(%d) = %v; want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestValidatorCheck(t *testing.T) {
	t.Run("passing check adds no error", func(t *testing.T) {
		v := Validator{}
		v.Check(true, "title", "Title is required")

		if !v.Valid() {
			t.Fatal("expected validator to be valid")
		}
	})

	t.Run("failing check adds field error", func(t *testing.T) {
		v := Validator{}
		v.Check(false, "title", "Title is required")

		if v.Valid() {
			t.Fatal("expected validator to be invalid")
		}
		if got := v.FieldErrors["title"]; got != "Title is required" {
			t.Fatalf("expected error message %q; got %q", "Title is required", got)
		}
	})

	t.Run("first error wins for same field", func(t *testing.T) {
		v := Validator{}
		v.Check(false, "title", "first error")
		v.Check(false, "title", "second error")

		if got := v.FieldErrors["title"]; got != "first error" {
			t.Fatalf("expected first error to win; got %q", got)
		}
	})

	t.Run("multiple fields track independently", func(t *testing.T) {
		v := Validator{}
		v.Check(false, "title", "Title is required")
		v.Check(false, "body", "Body is required")

		if len(v.FieldErrors) != 2 {
			t.Fatalf("expected 2 field errors; got %d", len(v.FieldErrors))
		}
	})
}

func TestValidatorAddFieldError(t *testing.T) {
	v := Validator{}

	// First call initializes the map.
	v.AddFieldError("email", "Email is required")
	if v.FieldErrors == nil {
		t.Fatal("expected FieldErrors to be initialized")
	}
	if got := v.FieldErrors["email"]; got != "Email is required" {
		t.Fatalf("expected %q; got %q", "Email is required", got)
	}

	// Second call for same field is a no-op.
	v.AddFieldError("email", "Should be ignored")
	if got := v.FieldErrors["email"]; got != "Email is required" {
		t.Fatalf("expected first message to persist; got %q", got)
	}
}
