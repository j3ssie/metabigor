// Package skill parses the SKILL.md bundles that ship inside the binary and
// teach a coding agent (Claude Code, Codex, or any agentskills.io-compatible
// agent) how to drive the metabigor CLI.
//
// A "skill" is a directory containing a SKILL.md: Markdown prose with a YAML
// frontmatter block declaring the skill's name and a one-line description.
// Optional reference files live under a references/ subdirectory and are pulled
// in only when the agent chooses to load them ("progressive disclosure").
package skill

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Meta is the frontmatter of a SKILL.md. Only the fields metabigor needs for
// listing and display are captured; the Markdown body is returned separately.
type Meta struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	License     string   `yaml:"license" json:"license,omitempty"`
	Tags        []string `yaml:"tags" json:"tags,omitempty"`
}

// frontmatterRe captures the YAML block at the very top of a file. CRLF is
// normalized to LF before it is applied.
var frontmatterRe = regexp.MustCompile(`(?s)\A---\s*\n(.*?)\n---\s*(?:\n|$)`)

// nameValidRe enforces the agentskills.io name rules: lowercase a-z0-9 with
// single hyphens between segments.
var nameValidRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Parse splits a SKILL.md blob into its frontmatter Meta and Markdown body.
// It returns a descriptive error when the frontmatter is missing or the name
// is invalid, so callers can skip a malformed bundle while loading the rest.
func Parse(raw []byte) (Meta, string, error) {
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")

	m := frontmatterRe.FindStringSubmatchIndex(content)
	if m == nil {
		return Meta{}, "", fmt.Errorf("missing YAML frontmatter (expected --- block at start of file)")
	}
	yamlBlock := content[m[2]:m[3]]
	body := strings.TrimSpace(content[m[1]:])

	var meta Meta
	if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
		return Meta{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}

	meta.Name = strings.TrimSpace(meta.Name)
	meta.Description = strings.TrimSpace(meta.Description)
	if meta.Name == "" {
		return Meta{}, "", fmt.Errorf("frontmatter.name is required")
	}
	if !nameValidRe.MatchString(meta.Name) || len(meta.Name) > 64 {
		return Meta{}, "", fmt.Errorf("frontmatter.name %q: must be lowercase a-z0-9 with single hyphens, <=64 chars", meta.Name)
	}
	if meta.Description == "" {
		return Meta{}, "", fmt.Errorf("frontmatter.description is required")
	}

	return meta, body, nil
}
