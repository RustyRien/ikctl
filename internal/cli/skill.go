package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rustyrien/ikctl/skills"
)

func newSkillCmd(f *Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Print or install the LLM agent skill for ikctl",
		Long: `ikctl ships an agent skill (SKILL.md plus reference files) that teaches LLM
agents how to operate InfraKitchen with ikctl. Print it, or install it into a
skills directory such as ~/.claude/skills or <project>/.claude/skills.`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}

	printCmd := &cobra.Command{
		Use:   "print [FILE]",
		Short: "Print SKILL.md (or a reference file, e.g. references/filters.md)",
		Example: `  ikctl skill print
  ikctl skill print references/filters.md`,
		Args: usageArgs(cobra.MaximumNArgs(1)),
		RunE: func(_ *cobra.Command, args []string) error {
			name := "SKILL.md"
			if len(args) == 1 {
				name = args[0]
			}
			data, err := fs.ReadFile(skills.FS, path.Join(skills.Name, name))
			if err != nil {
				return usageErrorf("unknown skill file %q", name)
			}
			_, err = f.IO.Out.Write(data)
			return err
		},
	}

	var (
		dir   string
		force bool
	)
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install the skill into a skills directory",
		Example: `  ikctl skill install                          # ~/.claude/skills/ikctl
  ikctl skill install --dir .claude/skills     # project-local`,
		Args: usageArgs(cobra.NoArgs),
		RunE: func(_ *cobra.Command, _ []string) error {
			if dir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				dir = filepath.Join(home, ".claude", "skills")
			}
			target := filepath.Join(dir, skills.Name)
			if _, err := os.Stat(target); err == nil && !force {
				return usageErrorf("%s already exists; pass --force to overwrite", target)
			}
			err := fs.WalkDir(skills.FS, skills.Name, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				dst := filepath.Join(dir, filepath.FromSlash(p))
				if d.IsDir() {
					return os.MkdirAll(dst, 0o755) //nolint:gosec // skill docs are not secret
				}
				data, err := fs.ReadFile(skills.FS, p)
				if err != nil {
					return err
				}
				return os.WriteFile(dst, data, 0o644) //nolint:gosec // skill docs are not secret
			})
			if err != nil {
				if errors.Is(err, fs.ErrPermission) {
					return fmt.Errorf("install skill: %w", err)
				}
				return err
			}
			fmt.Fprintf(f.IO.Out, "Installed skill to %s\n", target)
			return nil
		},
	}
	installCmd.Flags().StringVar(&dir, "dir", "", "skills directory (default ~/.claude/skills)")
	installCmd.Flags().BoolVar(&force, "force", false, "overwrite an existing installation")

	cmd.AddCommand(printCmd, installCmd)
	return cmd
}
