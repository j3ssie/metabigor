package skill

import "testing"

func TestParseValid(t *testing.T) {
	raw := []byte(`---
name: metabigor
description: >-
  Use when driving the metabigor CLI for OSINT recon:
  network ranges, certificates, related domains, and more.
license: MIT
tags:
  - osint
  - recon
---

# Metabigor

Body text here.
`)

	meta, body, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if meta.Name != "metabigor" {
		t.Errorf("Name = %q, want metabigor", meta.Name)
	}
	if meta.License != "MIT" {
		t.Errorf("License = %q, want MIT", meta.License)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "osint" {
		t.Errorf("Tags = %v, want [osint recon]", meta.Tags)
	}
	// Folded scalar collapses to a single line.
	if want := "Use when driving the metabigor CLI for OSINT recon:"; len(meta.Description) < len(want) {
		t.Errorf("Description = %q, too short", meta.Description)
	}
	if body == "" || body[0] != '#' {
		t.Errorf("body = %q, want it to start with the Markdown heading", body)
	}
}

func TestParseErrors(t *testing.T) {
	cases := map[string][]byte{
		"no frontmatter":    []byte("# Just markdown, no frontmatter\n"),
		"missing name":      []byte("---\ndescription: hi\n---\nbody\n"),
		"missing desc":      []byte("---\nname: metabigor\n---\nbody\n"),
		"invalid name":      []byte("---\nname: Meta_Bigor\ndescription: hi\n---\nbody\n"),
		"uppercase name":    []byte("---\nname: Metabigor\ndescription: hi\n---\nbody\n"),
		"broken yaml block": []byte("---\nname: [unclosed\ndescription: hi\n---\nbody\n"),
	}
	for name, raw := range cases {
		if _, _, err := Parse(raw); err == nil {
			t.Errorf("%s: expected an error, got nil", name)
		}
	}
}
