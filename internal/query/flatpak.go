package query

import "strings"

// ParseFlatpak parses `flatpak search` output: tab-separated columns
// Name, Description, ApplicationID, Version, Branch, Remotes (no header).
// Lines without at least the first four columns (e.g. "No matches found")
// are skipped.
func ParseFlatpak(out string) []Result {
	var results []Result
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 4 {
			continue
		}
		results = append(results, Result{
			Source:  "flatpak",
			Name:    strings.TrimSpace(f[2]),
			Version: strings.TrimSpace(f[3]),
			Desc:    strings.TrimSpace(f[1]),
		})
	}
	return results
}
