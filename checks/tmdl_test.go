package checks

import "testing"

func TestExtractTmdlBlock(t *testing.T) {
	input := "root\n  child one\n    grandchild\n\n  child two\nnext root"

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "blank query returns input",
			query: "  ",
			want:  input,
		},
		{
			name:  "missing query returns empty string",
			query: "missing",
			want:  "",
		},
		{
			name:  "extracts matching item block",
			query: "child one",
			want:  "  child one\n    grandchild",
		},
		{
			name:  "blank lines do not terminate block",
			query: "root",
			want:  "root\n  child one\n    grandchild\n\n  child two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTmdlBlock(input, tt.query)
			if got != tt.want {
				t.Fatalf("ExtractTmdlBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTmdlBlockHandlesTabIndentation(t *testing.T) {
	input := "item\n\tchild\n\t\tgrandchild\n\tsibling\nnext"

	got := ExtractTmdlBlock(input, "child")
	want := "\tchild\n\t\tgrandchild"
	if got != want {
		t.Fatalf("ExtractTmdlBlock() = %q, want %q", got, want)
	}
}
