package checks

import (
	"reflect"
	"strings"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestRunStdoutJqQuery(t *testing.T) {
	tests := []struct {
		name      string
		stdout    string
		test      api.StdoutJqTest
		variables map[string]string
		want      api.CLICommandJqOutput
		wantError string
	}{
		{
			name:   "queries json with interpolated query",
			stdout: `{"users":[{"name":"Lane"},{"name":"Theo"}]}`,
			test: api.StdoutJqTest{
				InputMode: "json",
				Query:     `.users[] | select(.name == "${name}") | .name`,
			},
			variables: map[string]string{"name": "Theo"},
			want: api.CLICommandJqOutput{
				Query:   `.users[] | select(.name == "Theo") | .name`,
				Results: []string{`"Theo"`},
			},
		},
		{
			name:   "queries jsonl as array",
			stdout: "{\"id\":1}\n{\"id\":2}\n",
			test: api.StdoutJqTest{
				InputMode: "jsonl",
				Query:     `.[].id`,
			},
			want: api.CLICommandJqOutput{
				Query:   `.[].id`,
				Results: []string{`1`, `2`},
			},
		},
		{
			name:   "returns parse error",
			stdout: `{not json}`,
			test: api.StdoutJqTest{
				InputMode: "json",
				Query:     `.name`,
			},
			want: api.CLICommandJqOutput{
				Query: `.name`,
			},
			wantError: "invalid character",
		},
		{
			name:   "returns jq error",
			stdout: `{"name":"Theo"}`,
			test: api.StdoutJqTest{
				InputMode: "json",
				Query:     `.name[`,
			},
			want: api.CLICommandJqOutput{
				Query: `.name[`,
				Error: "unexpected EOF",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runStdoutJqQuery(tt.stdout, tt.test, tt.variables)
			if tt.wantError != "" {
				if got.Query != tt.want.Query {
					t.Fatalf("Query = %q, want %q", got.Query, tt.want.Query)
				}
				if !strings.Contains(got.Error, tt.wantError) {
					t.Fatalf("expected error containing %q, got %q", tt.wantError, got.Error)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("runStdoutJqQuery() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseJqInputRejectsMultipleJSONValuesInJSONMode(t *testing.T) {
	_, err := parseJqInput("{\"id\":1}\n{\"id\":2}\n", "json")
	if err == nil {
		t.Fatal("expected error for multiple JSON values in json mode")
	}
	if err.Error() != "expected a single JSON value" {
		t.Fatalf("expected single-value error, got %q", err.Error())
	}
}

func TestFormatJqResults(t *testing.T) {
	got := formatJqResults([]any{"hello", float64(42), true, nil, map[string]any{"id": float64(1)}})
	want := []string{`"hello"`, `42`, `true`, `null`, `{"id":1}`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("formatJqResults() = %#v, want %#v", got, want)
	}
}

func TestFormatJqExpectedValueInterpolatesOnlyStrings(t *testing.T) {
	variables := map[string]string{"name": "Theo"}

	gotString := formatJqExpectedValue(api.JqExpectedResult{
		Type:  api.JqTypeString,
		Value: "hello ${name}",
	}, variables)
	if gotString != `"hello Theo"` {
		t.Fatalf("expected interpolated string value, got %q", gotString)
	}

	gotInt := formatJqExpectedValue(api.JqExpectedResult{
		Type:  api.JqTypeInt,
		Value: "${name}",
	}, variables)
	if gotInt != `"${name}"` {
		t.Fatalf("expected non-string jq type to avoid interpolation, got %q", gotInt)
	}
}

func TestValFromJqPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		jsn     string
		want    any
		wantErr string
	}{
		{
			name: "returns one value",
			path: `.token`,
			jsn:  `{"token":"abc123"}`,
			want: "abc123",
		},
		{
			name:    "errors on missing value",
			path:    `.missing`,
			jsn:     `{"token":"abc123"}`,
			wantErr: "value not found",
		},
		{
			name:    "errors on multiple values",
			path:    `.items[].id`,
			jsn:     `{"items":[{"id":1},{"id":2}]}`,
			wantErr: "invalid number of values found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := valFromJqPath(tt.path, tt.jsn)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("valFromJqPath() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
