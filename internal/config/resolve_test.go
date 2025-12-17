package config

import (
	"testing"
)

func TestNewResolver(t *testing.T) {
	t.Setenv("TEST_VAR", "test_value")

	r := NewResolver()
	if r == nil {
		t.Fatal("NewResolver returned nil")
	}

	if r.env["TEST_VAR"] != "test_value" {
		t.Errorf("expected TEST_VAR=test_value, got %q", r.env["TEST_VAR"])
	}
}

func TestNewResolverWithEnv(t *testing.T) {
	env := map[string]string{
		"CUSTOM_VAR": "custom_value",
	}

	r := NewResolverWithEnv(env)
	if r == nil {
		t.Fatal("NewResolverWithEnv returned nil")
	}

	if r.env["CUSTOM_VAR"] != "custom_value" {
		t.Errorf("expected CUSTOM_VAR=custom_value, got %q", r.env["CUSTOM_VAR"])
	}
}

func TestResolver_Resolve(t *testing.T) {
	env := map[string]string{
		"API_KEY":    "secret123",
		"BASE_URL":   "https://api.example.com",
		"EMPTY_VAR":  "",
		"MULTI_WORD": "hello world",
	}
	r := NewResolverWithEnv(env)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "no variables",
			input:   "plain text",
			want:    "plain text",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "simple $VAR",
			input:   "$API_KEY",
			want:    "secret123",
			wantErr: false,
		},
		{
			name:    "braced ${VAR}",
			input:   "${API_KEY}",
			want:    "secret123",
			wantErr: false,
		},
		{
			name:    "variable in text",
			input:   "key=$API_KEY",
			want:    "key=secret123",
			wantErr: false,
		},
		{
			name:    "braced variable in text",
			input:   "key=${API_KEY}",
			want:    "key=secret123",
			wantErr: false,
		},
		{
			name:    "multiple variables",
			input:   "$BASE_URL/v1?key=$API_KEY",
			want:    "https://api.example.com/v1?key=secret123",
			wantErr: false,
		},
		{
			name:    "mixed brace styles",
			input:   "${BASE_URL}/v1?key=$API_KEY",
			want:    "https://api.example.com/v1?key=secret123",
			wantErr: false,
		},
		{
			name:    "undefined variable",
			input:   "$UNDEFINED_VAR",
			want:    "",
			wantErr: true,
		},
		{
			name:    "undefined braced variable",
			input:   "${UNDEFINED_VAR}",
			want:    "",
			wantErr: true,
		},
		{
			name:    "multiple undefined variables",
			input:   "$UNDEFINED1 and $UNDEFINED2",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty variable value",
			input:   "$EMPTY_VAR",
			want:    "",
			wantErr: false,
		},
		{
			name:    "variable with spaces in value",
			input:   "$MULTI_WORD",
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "dollar sign without variable",
			input:   "cost is $50",
			want:    "cost is $50",
			wantErr: false,
		},
		{
			name:    "adjacent variables",
			input:   "${API_KEY}${BASE_URL}",
			want:    "secret123https://api.example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolver_MustResolve(t *testing.T) {
	env := map[string]string{
		"EXISTING_VAR": "value",
	}
	r := NewResolverWithEnv(env)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "existing variable",
			input: "$EXISTING_VAR",
			want:  "value",
		},
		{
			name:  "undefined variable returns empty",
			input: "$UNDEFINED",
			want:  "",
		},
		{
			name:  "plain text",
			input: "plain",
			want:  "plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.MustResolve(tt.input)
			if got != tt.want {
				t.Errorf("MustResolve() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolver_Resolve_VariableNamePatterns(t *testing.T) {
	env := map[string]string{
		"VAR":          "a",
		"VAR1":         "b",
		"VAR_NAME":     "c",
		"_VAR":         "d",
		"VAR123":       "e",
		"MY_VAR_2":     "f",
		"A":            "g",
		"UPPER_CASE":   "h",
		"camelCase":    "i",
		"MixedCase123": "j",
	}
	r := NewResolverWithEnv(env)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple name", "$VAR", "a"},
		{"name with number", "$VAR1", "b"},
		{"name with underscore", "$VAR_NAME", "c"},
		{"name starting with underscore", "$_VAR", "d"},
		{"name with numbers", "$VAR123", "e"},
		{"complex name", "$MY_VAR_2", "f"},
		{"single char", "$A", "g"},
		{"upper case", "$UPPER_CASE", "h"},
		{"camel case", "$camelCase", "i"},
		{"mixed case with numbers", "$MixedCase123", "j"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Resolve(tt.input)
			if err != nil {
				t.Errorf("Resolve() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve() = %q, want %q", got, tt.want)
			}
		})
	}
}
