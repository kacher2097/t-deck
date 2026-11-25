package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"t-deck/internal/fsutil"
)

func syncFTSTx(tx *sql.Tx, projectID, name, summary, tagsSpace, techStack string) error {
	if _, err := tx.Exec(`DELETE FROM projects_fts WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("clear fts: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT INTO projects_fts (project_id, name, summary, tags, tech_stack) VALUES (?, ?, ?, ?, ?)`,
		projectID, name, summary, tagsSpace, techStack,
	); err != nil {
		return fmt.Errorf("insert fts: %w", err)
	}
	return nil
}

// buildFTSQuery tokenizes user input safely for FTS5 MATCH with restrained prefix matching.
func buildFTSQuery(raw string) string {
	const maxFTSTokens = 16
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return r
		}
		return ' '
	}, raw)
	fields := strings.Fields(cleaned)
	if len(fields) == 0 {
		return ""
	}
	if len(fields) > maxFTSTokens {
		fields = fields[:maxFTSTokens]
	}

	// When multiple tokens exist (e.g. "t-deck" -> ["t", "deck"]), drop noisy 1-rune tokens
	// unless every token is 1 rune long.
	filtered := make([]string, 0, len(fields))
	for _, f := range fields {
		if utf8.RuneCountInString(f) >= 2 {
			filtered = append(filtered, f)
		}
	}
	if len(filtered) == 0 {
		// Single or all 1-rune tokens are too broad for FTS prefix search; fall back to substring filter.
		return ""
	}

	terms := make([]string, 0, len(filtered))
	for _, f := range filtered {
		if utf8.RuneCountInString(f) >= 3 {
			terms = append(terms, `"`+f+`"*`)
		} else {
			// Exact token match for 2-rune words ("go", "ai", "r2") to avoid matching hundreds of prefixes.
			terms = append(terms, `"`+f+`"`)
		}
	}
	return strings.Join(terms, " AND ")
}

func decodeTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return []string{}
	}
	return normalizeTags(tags)
}

func normalizeTags(tags []string) []string {
	cleaned := make([]string, 0, len(tags))
	for _, t := range tags {
		cleaned = append(cleaned, strings.TrimPrefix(strings.TrimSpace(t), "#"))
	}
	return fsutil.UniqueStrings(cleaned)
}

