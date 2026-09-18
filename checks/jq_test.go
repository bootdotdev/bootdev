package checks

import (
	"reflect"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestRunStdoutJqQuery(t *testing.T) {
	tests := []struct {
		name      string
		stdout    string
		test      api.StdoutJqTest
		want      api.CLICommandJqOutput
		wantError bool
	}{
		{
			name: "queries JSON with comments using a literal query",
			stdout: `{
				// Users to query
				"users": [/* users */ {"name":"Lane"},{"name":"${name}",},],
			}`,
			test: api.StdoutJqTest{
				InputMode: "json",
				Query:     `.users[] | select(.name == "${name}") | .name`,
			},
			want: api.CLICommandJqOutput{
				Query:   `.users[] | select(.name == "${name}") | .name`,
				Results: []string{`"${name}"`},
			},
		},
		{
			name:   "default mode accepts comments and trailing commas",
			stdout: `{"name": /* user */ "Boots",} // final comment without newline`,
			test:   api.StdoutJqTest{Query: `.name`},
			want: api.CLICommandJqOutput{
				Query:   `.name`,
				Results: []string{`"Boots"`},
			},
		},
		{
			name:   "preserves large integers",
			stdout: `{"id":9007199254740993,}`,
			test:   api.StdoutJqTest{InputMode: "json", Query: `.id`},
			want: api.CLICommandJqOutput{
				Query:   `.id`,
				Results: []string{`9007199254740993`},
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
			stdout: `{"name":"Boots"} /* unterminated`,
			test: api.StdoutJqTest{
				InputMode: "json",
				Query:     `.name`,
			},
			want: api.CLICommandJqOutput{
				Query: `.name`,
			},
			wantError: true,
		},
		{
			name:   "returns jq error",
			stdout: `{"name":"Kaladin"}`,
			test: api.StdoutJqTest{
				InputMode: "json",
				Query:     `.name[`,
			},
			want: api.CLICommandJqOutput{
				Query: `.name[`,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runStdoutJqQuery(tt.stdout, tt.test)
			if tt.wantError {
				if got.Query != tt.want.Query {
					t.Fatalf("Query = %q, want %q", got.Query, tt.want.Query)
				}
				if got.Error == "" {
					t.Fatal("expected an error")
				}
				if len(got.Results) != 0 {
					t.Fatalf("expected no results on error, got %v", got.Results)
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
