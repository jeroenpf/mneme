package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeroenpf/mneme/internal/backup"
	"github.com/jeroenpf/mneme/internal/config"
	"github.com/jeroenpf/mneme/internal/migrations"
	"github.com/jeroenpf/mneme/internal/store"
)

// newExportCmd builds `mneme export <file>`: dump all local knowledge to a
// portable JSON archive for backup. Backend-agnostic (reads through the store),
// so the archive restores into either backend. `-` writes to stdout.
func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <file>",
		Short: "Export all local knowledge to a portable JSON backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadWithFlags(cmd.Flags())
			if err != nil {
				return err
			}
			return runExport(cmd, cfg, args[0])
		},
	}
	cmd.Flags().String("dsn", "", "storage DSN (sqlite:// file or postgres:// URL); overrides config/env")
	return cmd
}

func runExport(cmd *cobra.Command, cfg *config.Config, path string) error {
	ctx := cmd.Context()
	if err := migrations.Up(cfg.DSN); err != nil {
		return fmt.Errorf("prepare schema: %w", err)
	}
	st, err := store.New(ctx, cfg.DSN)
	if err != nil {
		return err
	}
	defer st.Close()

	arch, err := backup.Export(ctx, st)
	if err != nil {
		return err
	}

	if path == "-" {
		return arch.Write(cmd.OutOrStdout())
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := arch.Write(f); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// Round-trip self-check: re-read what we just wrote and confirm it matches
	// the in-memory archive, so a corrupt or truncated backup is caught now.
	if err := verifyWritten(path, arch); err != nil {
		return fmt.Errorf("backup written but failed verification: %w", err)
	}
	cmd.Printf("exported %d document(s), %d decision(s), %d snippet(s), %d journal entr(ies), %d solution(s), %d memory, %d env to %s (verified)\n",
		len(arch.Documents), len(arch.Decisions), len(arch.Snippets), len(arch.Journal), len(arch.Solutions), len(arch.Memory), len(arch.Env), path)
	return nil
}

// verifyWritten re-reads a written archive file and verifies it round-trips to
// the source archive by content.
func verifyWritten(path string, src *backup.Archive) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	back, err := backup.Read(f)
	if err != nil {
		return err
	}
	return backup.Verify(src, back)
}

// newImportCmd builds `mneme import <file>`: restore a JSON backup into the
// configured store. Rows that already exist (project slug, document id) are
// skipped, so importing into a live store is safe.
func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Restore local knowledge from a JSON backup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadWithFlags(cmd.Flags())
			if err != nil {
				return err
			}
			return runImport(cmd, cfg, args[0])
		},
	}
	cmd.Flags().String("dsn", "", "storage DSN (sqlite:// file or postgres:// URL); overrides config/env")
	return cmd
}

func runImport(cmd *cobra.Command, cfg *config.Config, path string) error {
	ctx := cmd.Context()
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	arch, err := backup.Read(f)
	if err != nil {
		return err
	}

	if err := migrations.Up(cfg.DSN); err != nil {
		return fmt.Errorf("prepare schema: %w", err)
	}
	st, err := store.New(ctx, cfg.DSN)
	if err != nil {
		return err
	}
	defer st.Close()

	res, err := backup.Import(ctx, st, arch)
	if err != nil {
		return err
	}
	for _, kind := range []string{"projects", "documents", "decisions", "snippets", "journal", "solutions", "memory", "env"} {
		created, skipped := res.Created[kind], res.Skipped[kind]
		if created == 0 && skipped == 0 {
			continue
		}
		cmd.Printf("  %s: created %d", kind, created)
		if skipped > 0 {
			cmd.Printf(" (skipped %d already present)", skipped)
		}
		cmd.Println()
	}
	cmd.Println("import complete")
	return nil
}
