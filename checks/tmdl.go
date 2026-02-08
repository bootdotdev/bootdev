package checks

import "strings"

const tabWidth = 4

// Find the first line whose left-trimmed text has prefix `query`, and return the
// "item block": that line plus following lines until the next non-empty line with
// indent <= the matched line's indent
func ExtractTmdlBlock(input, query string) string {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return input
	}

	lines := strings.Split(input, "\n")

	found := false
	baseIndent := 0

	var out strings.Builder

	for _, line := range lines {
		if !found {
			if hasLeftTrimmedPrefix(line, trimmedQuery) {
				found = true
				baseIndent = indentWidth(line)

				out.WriteString(line)
				out.WriteByte('\n')
			}

			continue
		}

		// Don't let blank lines terminate the block
		if strings.TrimSpace(line) != "" && indentWidth(line) <= baseIndent {
			break
		}

		out.WriteString(line)
		out.WriteByte('\n')
	}

	if !found {
		return ""
	}

	return strings.TrimRight(out.String(), " \n\t\r")
}

func hasLeftTrimmedPrefix(line, prefix string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trimmed, prefix)
}

func indentWidth(line string) int {
	w := 0

	for _, r := range line {
		switch r {
		case ' ':
			w++
		case '\t':
			w += tabWidth
		default:
			return w
		}
	}

	return w
}
