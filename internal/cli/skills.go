package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/skill"
	"github.com/j3ssie/metabigor/public"
	"github.com/spf13/cobra"
)

// embedSkillsRoot is the directory inside public.SkillsFS that holds the
// coding-agent skill bundles.
const embedSkillsRoot = "skills"

// defaultSkill is installed when `skills install` runs without a name.
const defaultSkill = "metabigor"

// Install destinations, relative to a project directory or the home directory.
const (
	claudeSkillsSubdir = ".claude/skills"
	agentsSkillsSubdir = ".agents/skills"
)

var skillsOpts struct {
	Full  bool
	All   bool
	Agent string
	Scope string
	Dir   string
	Force bool
}

// bundledSkill is a parsed skill bundle shipped inside the binary.
type bundledSkill struct {
	Name        string   // directory / frontmatter name (e.g. "metabigor")
	Description string   // frontmatter description
	EmbedDir    string   // path inside public.SkillsFS (e.g. "skills/metabigor")
	References  []string // reference paths relative to EmbedDir (e.g. "references/recipes.md")
}

const skillsLong = `List, print, and install the coding-agent skills bundled with metabigor.

The bundled skills teach an AI coding agent (Claude Code, Codex, or any
agentskills.io-compatible agent) how to drive the metabigor CLI. Because the
content is embedded in the binary, it always matches the installed version —
prefer 'metabigor skills install' over copying stale files by hand.

Running 'metabigor skills' with no subcommand lists the bundled skills.`

var skillsCmd = &cobra.Command{
	Use:     "skills [subcommand]",
	Aliases: []string{"skill"},
	Short:   "List, print, and install bundled coding-agent skills",
	Long:    skillsLong,
	GroupID: groupUtil,
	Example: examples(
		"list the bundled skills", "metabigor skills",
		"print a skill and its references", "metabigor skills get --full",
		"install into Claude Code (this project)", "metabigor skills install",
		"install globally for Codex/agents", "metabigor skills install --agent agents --scope global",
	),
	Args: cobra.ArbitraryArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runSkillsList()
	},
}

var skillsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all bundled skills",
	Args:    cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runSkillsList()
	},
}

var skillsGetCmd = &cobra.Command{
	Use:   "get [name...]",
	Short: "Print a skill's content to stdout",
	Long: `Print a skill's SKILL.md to stdout. Pass --full to also include its
reference files, or --all to print every bundled skill. Use -o to also write
the content to a file.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(_ *cobra.Command, args []string) error {
		return runSkillsGet(args)
	},
}

const skillsInstallLong = `Copy one or more bundled skills into a coding agent's skills directory so the
agent can auto-trigger on them.

With no name, installs the '` + defaultSkill + `' skill. Pass --all to install every
bundle, or name specific skills.

Destination is chosen from --agent and --scope:

  --agent claude          .claude/skills/   (project)   ~/.claude/skills/   (global)
  --agent codex|agents    .agents/skills/   (project)   ~/.agents/skills/   (global)

An already-installed skill is skipped unless --force is given. --dir overrides
the computed destination entirely.`

var skillsInstallCmd = &cobra.Command{
	Use:   "install [name...]",
	Short: "Install skill bundle(s) into a coding agent's skills directory",
	Long:  skillsInstallLong,
	Example: examples(
		"install metabigor for Claude Code in this project", "metabigor skills install",
		"install every bundle globally for Codex", "metabigor skills install --all --agent agents --scope global",
		"overwrite an existing copy", "metabigor skills install --force",
		"install into a custom directory", "metabigor skills install --dir ./my-skills",
	),
	Args: cobra.ArbitraryArgs,
	RunE: func(_ *cobra.Command, args []string) error {
		return runSkillsInstall(args)
	},
}

func init() {
	rootCmd.AddCommand(skillsCmd)
	skillsCmd.AddCommand(skillsListCmd, skillsGetCmd, skillsInstallCmd)

	skillsGetCmd.Flags().BoolVar(&skillsOpts.Full, "full", false, "Include reference files, not just SKILL.md")
	skillsGetCmd.Flags().BoolVar(&skillsOpts.All, "all", false, "Print every bundled skill")

	skillsInstallCmd.Flags().StringVar(&skillsOpts.Agent, "agent", "claude", "Target coding agent: claude, codex, or agents")
	skillsInstallCmd.Flags().StringVar(&skillsOpts.Scope, "scope", "project", "Install scope: project (current folder) or global (home dir)")
	skillsInstallCmd.Flags().StringVar(&skillsOpts.Dir, "dir", "", "Override the destination directory (skips --agent/--scope resolution)")
	skillsInstallCmd.Flags().BoolVar(&skillsOpts.All, "all", false, "Install every bundled skill")
	skillsInstallCmd.Flags().BoolVar(&skillsOpts.Force, "force", false, "Overwrite an already-installed skill")
}

// loadBundledSkills discovers and parses every skill bundle embedded under
// public.SkillsFS/skills. A "bundle" is a directory containing a SKILL.md.
func loadBundledSkills() ([]bundledSkill, error) {
	entries, err := fs.ReadDir(public.SkillsFS, embedSkillsRoot)
	if err != nil {
		return nil, fmt.Errorf("read embedded skills: %w", err)
	}

	var out []bundledSkill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		embedDir := embedSkillsRoot + "/" + e.Name()
		raw, readErr := public.SkillsFS.ReadFile(embedDir + "/SKILL.md")
		if readErr != nil {
			continue // not a skill bundle
		}

		// The directory name is the bundle's canonical identity: it names the
		// install destination and is what CLI args reference. Only the
		// description is pulled from frontmatter, for display. Bundles are
		// authored in-tree, so a parse failure lists without a description
		// rather than dropping the bundle.
		desc := ""
		if meta, _, perr := skill.Parse(raw); perr == nil {
			desc = meta.Description
		}

		out = append(out, bundledSkill{
			Name:        e.Name(),
			Description: desc,
			EmbedDir:    embedDir,
			References:  listBundleReferences(embedDir),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// listBundleReferences returns the reference files under <embedDir>/references,
// as paths relative to embedDir, sorted.
func listBundleReferences(embedDir string) []string {
	entries, err := fs.ReadDir(public.SkillsFS, embedDir+"/references")
	if err != nil {
		return nil
	}
	var refs []string
	for _, e := range entries {
		if !e.IsDir() {
			refs = append(refs, "references/"+e.Name())
		}
	}
	sort.Strings(refs)
	return refs
}

// findBundle returns the bundle with the given name (case-insensitive).
func findBundle(skills []bundledSkill, name string) (bundledSkill, bool) {
	for _, s := range skills {
		if strings.EqualFold(s.Name, name) {
			return s, true
		}
	}
	return bundledSkill{}, false
}

// bundleNames joins bundle names for error messages.
func bundleNames(skills []bundledSkill) string {
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name
	}
	return strings.Join(names, ", ")
}

// resolveNamedTargets maps explicit skill names to bundles, erroring on the
// first unknown name. Shared by get and install.
func resolveNamedTargets(skills []bundledSkill, names []string) ([]bundledSkill, error) {
	targets := make([]bundledSkill, 0, len(names))
	for _, n := range names {
		b, ok := findBundle(skills, n)
		if !ok {
			return nil, fmt.Errorf("unknown skill %q (available: %s)", n, bundleNames(skills))
		}
		targets = append(targets, b)
	}
	return targets, nil
}

// skillRecord renders one bundled skill through output.Record, so `skills list`
// honors -f/--format like every other command.
type skillRecord struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	EmbedPath   string   `json:"embed_path"`
	References  []string `json:"references,omitempty"`
}

func (s skillRecord) Text() []string {
	lines := []string{s.Name}
	if s.Description != "" {
		lines = append(lines, "  "+s.Description)
	}
	if len(s.References) > 0 {
		lines = append(lines, "  references: "+strings.Join(s.References, ", "))
	}
	return append(lines, "") // trailing blank separates records
}

func (s skillRecord) Flat() []string { return []string{s.Name} }

func (s skillRecord) CSV() (header []string, rows [][]string) {
	return []string{"name", "description", "references", "embed_path"},
		[][]string{{s.Name, s.Description, strings.Join(s.References, " "), s.EmbedPath}}
}

func runSkillsList() error {
	skills, err := loadBundledSkills()
	if err != nil {
		return err
	}

	format, err := output.ParseFormat(opt.Format)
	if err != nil {
		return err
	}
	w, err := output.NewWriter(opt.Output, format, opt.Append)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := w.Close(); closeErr != nil {
			output.Error("close output file: %v", closeErr)
		}
	}()

	for _, s := range skills {
		w.Write(skillRecord{
			Name:        s.Name,
			Description: s.Description,
			EmbedPath:   s.EmbedDir,
			References:  s.References,
		})
	}

	if len(skills) == 0 {
		output.Warn("No skills are bundled with this build")
		return nil
	}
	output.Info("Print one with: metabigor skills get %s --full", skills[0].Name)
	output.Info("Install it with: metabigor skills install %s", skills[0].Name)
	return nil
}

// writeBody writes raw bytes to w, appending a newline only when the content
// does not already end with one.
func writeBody(w io.Writer, data []byte) {
	_, _ = w.Write(data)
	if len(data) > 0 && data[len(data)-1] != '\n' {
		_, _ = fmt.Fprintln(w)
	}
}

func runSkillsGet(names []string) error {
	skills, err := loadBundledSkills()
	if err != nil {
		return err
	}

	var targets []bundledSkill
	switch {
	case skillsOpts.All:
		targets = skills
	case len(names) == 0:
		return fmt.Errorf("skills get: provide a skill name or --all (available: %s)", bundleNames(skills))
	default:
		if targets, err = resolveNamedTargets(skills, names); err != nil {
			return err
		}
	}

	// Mirror the "-o also writes to a file" contract: content goes to stdout
	// and, when -o is set, to the file too.
	w := io.Writer(os.Stdout)
	if opt.Output != "" {
		flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		if opt.Append {
			flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
		}
		f, openErr := os.OpenFile(opt.Output, flags, 0o644)
		if openErr != nil {
			return fmt.Errorf("open output file %s: %w", opt.Output, openErr)
		}
		defer func() { _ = f.Close() }()
		w = io.MultiWriter(os.Stdout, f)
	}

	for i, b := range targets {
		if len(targets) > 1 {
			if i > 0 {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "===== %s =====\n\n", b.Name)
		}
		body, readErr := public.SkillsFS.ReadFile(b.EmbedDir + "/SKILL.md")
		if readErr != nil {
			return fmt.Errorf("read %s: %w", b.Name, readErr)
		}
		writeBody(w, body)

		if skillsOpts.Full {
			for _, ref := range b.References {
				refBody, refErr := public.SkillsFS.ReadFile(b.EmbedDir + "/" + ref)
				if refErr != nil {
					continue
				}
				fmt.Fprintf(w, "\n\n----- %s/%s -----\n\n", b.Name, ref)
				writeBody(w, refBody)
			}
		}
	}
	return nil
}

// skillsInstallBaseDir resolves the parent directory skill bundles install
// into, from --agent and --scope. Returns an absolute path.
func skillsInstallBaseDir(agent, scope string) (string, error) {
	var sub string
	switch strings.ToLower(agent) {
	case "claude":
		sub = claudeSkillsSubdir
	case "codex", "agents":
		sub = agentsSkillsSubdir
	default:
		return "", fmt.Errorf("unknown --agent %q (want: claude, codex, or agents)", agent)
	}

	switch strings.ToLower(scope) {
	case "project":
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working directory: %w", err)
		}
		return filepath.Join(cwd, sub), nil
	case "global":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, sub), nil
	default:
		return "", fmt.Errorf("unknown --scope %q (want: project or global)", scope)
	}
}

func runSkillsInstall(names []string) error {
	skills, err := loadBundledSkills()
	if err != nil {
		return err
	}

	var targets []bundledSkill
	switch {
	case skillsOpts.All:
		targets = skills
	case len(names) == 0:
		b, ok := findBundle(skills, defaultSkill)
		if !ok {
			return fmt.Errorf("default skill %q is not bundled (available: %s)", defaultSkill, bundleNames(skills))
		}
		targets = []bundledSkill{b}
	default:
		if targets, err = resolveNamedTargets(skills, names); err != nil {
			return err
		}
	}

	baseDir := skillsOpts.Dir
	if baseDir == "" {
		if baseDir, err = skillsInstallBaseDir(skillsOpts.Agent, skillsOpts.Scope); err != nil {
			return err
		}
	} else if abs, absErr := filepath.Abs(baseDir); absErr == nil {
		baseDir = abs
	}

	installed, skipped := 0, 0
	for _, b := range targets {
		dest := filepath.Join(baseDir, b.Name)
		if _, statErr := os.Stat(filepath.Join(dest, "SKILL.md")); statErr == nil && !skillsOpts.Force {
			output.Warn("%s already installed at %s (use --force to overwrite)", b.Name, dest)
			skipped++
			continue
		}
		if copyErr := copyEmbeddedSkillBundle(b.EmbedDir, dest); copyErr != nil {
			return fmt.Errorf("install %s: %w", b.Name, copyErr)
		}
		output.Good("Installed %s -> %s", b.Name, dest)
		installed++
	}

	output.Info("%d installed, %d skipped. Your agent auto-triggers on these when you mention metabigor.", installed, skipped)
	return nil
}

// copyEmbeddedSkillBundle recursively copies an embedded bundle directory to
// destRoot on disk, creating parent directories as needed.
func copyEmbeddedSkillBundle(embedDir, destRoot string) error {
	return fs.WalkDir(public.SkillsFS, embedDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(embedDir, path)
		if relErr != nil {
			return relErr
		}
		dest := filepath.Join(destRoot, rel)

		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		data, readErr := public.SkillsFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if mkErr := os.MkdirAll(filepath.Dir(dest), 0o755); mkErr != nil {
			return mkErr
		}
		return os.WriteFile(dest, data, 0o644)
	})
}
