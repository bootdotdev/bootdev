package checks

import "testing"

func TestExtractTmdlBlock(t *testing.T) {
	input := "root\n  child one\n    grandchild\n\n  child two\nnext root"
	for _, tt := range []struct {
		name, input, query, want string
	}{
		{"blank query", input, "  ", input},
		{"missing query", input, "missing", ""},
		{"nested block", input, "child one", "  child one\n    grandchild"},
		{"blank lines inside block", input, "root", "root\n  child one\n    grandchild\n\n  child two"},
		{"tabs", "item\n\tchild\n\t\tgrandchild\n\tsibling\nnext", "child", "\tchild\n\t\tgrandchild"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractTmdlBlock(tt.input, tt.query); got != tt.want {
				t.Fatalf("ExtractTmdlBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}
