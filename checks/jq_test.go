package checks

import (
	"slices"
	"testing"

	api "github.com/bootdotdev/bootdev/client"
)

func TestRunStdoutJqQuery(t *testing.T) {
	for _, tt := range []struct {
		name, mode, stdout, query string
		want                      []string
		wantError                 bool
	}{
		{
			name: "JSON comments and literal query", mode: "json",
			stdout: `{
				// Users to query
				"users": [/* users */ {"name":"Lane"},{"name":"${name}",},],
			}`,
			query: `.users[] | select(.name == "${name}") | .name`,
			want:  []string{`"${name}"`},
		},
		{
			name:   "default mode and final comment without newline",
			stdout: `{"name": "Boots",} // final comment`, query: `.name`,
			want: []string{`"Boots"`},
		},
		{
			name: "preserves large integers", mode: "json",
			stdout: `{"id":9007199254740993,}`, query: `.id`,
			want: []string{`9007199254740993`},
		},
		{
			name: "JSONL as array", mode: "jsonl",
			stdout: "{\"id\":1}\n{\"id\":2}\n", query: `.[].id`,
			want: []string{`1`, `2`},
		},
		{
			name: "unterminated comment", stdout: `{"name":"Boots"} /* unterminated`,
			query: `.name`, wantError: true,
		},
		{
			name: "invalid query", stdout: `{"name":"Boots"}`,
			query: `.name[`, wantError: true,
		},
		{
			name: "multiple JSON values", mode: "json", stdout: `{"id":1} {"id":2}`,
			query: `.id`, wantError: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := runStdoutJqQuery(tt.stdout, api.StdoutJqTest{InputMode: tt.mode, Query: tt.query})
			if got.Query != tt.query || (got.Error != "") != tt.wantError || !slices.Equal(got.Results, tt.want) {
				t.Fatalf("got %#v; want query %q, results %v, error %t", got, tt.query, tt.want, tt.wantError)
			}
		})
	}
}

func TestValFromJqPathRejectsMultipleValues(t *testing.T) {
	_, err := valFromJqPath(`.items[].id`, `{"items":[{"id":1},{"id":2}]}`)
	if err == nil {
		t.Fatal("expected error for multiple values")
	}
}
