package query

import "strings"

// ParsePacman parses `pacman -Ss` output. Each hit is a header line
// "repo/name version [tags]" optionally followed by an indented description.
func ParsePacman(out string) []Result {
	var results []Result
	lines := strings.Split(out, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue // blank or description line (consumed with its header)
		}
		r, ok := parsePacmanHeader(line)
		if !ok {
			continue
		}
		if i+1 < len(lines) && len(lines[i+1]) > 0 && (lines[i+1][0] == ' ' || lines[i+1][0] == '\t') {
			r.Desc = strings.TrimSpace(lines[i+1])
			i++
		}
		results = append(results, r)
	}
	return results
}

// parsePacmanHeader parses "repo/name version [tags...]".
func parsePacmanHeader(line string) (Result, bool) {
	slash := strings.IndexByte(line, '/')
	if slash < 0 {
		return Result{}, false
	}
	repo := line[:slash]
	fields := strings.Fields(line[slash+1:])
	if len(fields) < 2 {
		return Result{}, false
	}
	return Result{
		Source:    repo,
		Name:      fields[0],
		Version:   fields[1],
		Installed: strings.Contains(line, "[installed"),
	}, true
}
