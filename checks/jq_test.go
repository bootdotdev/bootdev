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
		variables map[string]string
		want      api.CLICommandJqOutput
		wantError bool
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
			got := runStdoutJqQuery(tt.stdout, tt.test, tt.variables)
			if tt.wantError {
				if got.Query != tt.want.Query {
					t.Fatalf("Query = %q, want %q", got.Query, tt.want.Query)
				}
				if got.Error == "" {
					t.Fatal("expected an error")
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

func TestRunStdoutJqQueryJSONC(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		query  string
		want   []string
	}{
		{"queries normalized JSONC", `{
			// Users to query
			"users": [/* primary user */ {"name":"Boots",},],
		}`, `.users[].name`, []string{`"Boots"`}},
		{"large integer", `{"id":9007199254740993,}`, `.id`, []string{`9007199254740993`}},
		{"trailing line comment", `{"name":"Boots"} // comment`, `.name`, []string{`"Boots"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runStdoutJqQuery(tt.stdout, api.StdoutJqTest{InputMode: "jsonc", Query: tt.query}, nil)
			if got.Error != "" {
				t.Fatalf("unexpected error: %s", got.Error)
			}
			if !reflect.DeepEqual(got.Results, tt.want) {
				t.Fatalf("results = %v, want %v", got.Results, tt.want)
			}
		})
	}
}

func TestRunStdoutJqQueryReturnsJSONCParseError(t *testing.T) {
	got := runStdoutJqQuery(`{"name":"Boots"} /* unterminated`, api.StdoutJqTest{InputMode: "jsonc", Query: "."}, nil)
	if got.Error == "" || len(got.Results) != 0 || got.Query != "." {
		t.Fatalf("expected query with parse error and no results, got %#v", got)
	}
}

func TestParseJqInputJSONCModeIsolation(t *testing.T) {
	for _, mode := range []string{"json", "jsonl"} {
		for _, stdout := range []string{`{/* comment */ "name":"Boots"}`, `{"name":"Boots",}`} {
			t.Run(mode+stdout, func(t *testing.T) {
				if _, err := parseJqInput(stdout, mode); err == nil {
					t.Fatal("expected strict input parsing to reject JSONC")
				}
			})
		}
	}
	t.Run("normalized input mode", func(t *testing.T) {
		got, err := parseJqInput(`/* comment */ true`, " JSONC \t")
		if err != nil || got != true {
			t.Fatalf("got %v, %v; want true, nil", got, err)
		}
	})
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
