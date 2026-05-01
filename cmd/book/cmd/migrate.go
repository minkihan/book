package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"book/internal"
	"book/internal/migrate"
	"book/internal/plan"
)

var dryRun bool

// migrateCmd represents the migrate command.
//
// Korean: migrate 커맨드를 나타낸다.
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Add frontmatter to existing plans",
	Long: `Migrate existing plans by inserting YAML frontmatter.
Data is extracted from TODO.md and each plan's readme.md.
Use --dry-run to preview changes without modifying files.

Korean: 기존 플랜에 YAML frontmatter를 삽입한다.
TODO.md와 각 플랜의 readme.md에서 데이터를 추출한다.
--dry-run으로 파일 수정 없이 변경 내용을 미리 볼 수 있다.`,
	RunE: runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without modifying files")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	root, err := internal.BookRoot()
	if err != nil {
		return err
	}

	// parse TODO.md
	todoPath := filepath.Join(root, "TODO.md")
	todoEntries, err := migrate.ParseTodoMD(todoPath)
	if err != nil {
		return fmt.Errorf("TODO.md 파싱 실패: %w", err)
	}

	// list all plans
	plans, err := plan.ListPlanDirs(root)
	if err != nil {
		return err
	}

	migrated := 0
	skipped := 0
	failed := 0

	for i := range plans {
		p := &plans[i]
		meta, err := migrate.MigratePlan(p, todoEntries)
		if err != nil {
			fmt.Printf("  FAIL  P%03d %s: %v\n", p.Number, p.Slug, err)
			failed++
			continue
		}
		if meta == nil {
			skipped++
			continue
		}

		if dryRun {
			fmt.Printf("  [DRY] P%03d %-40s status=%-9s project=%-8s",
				p.Number, p.Slug, meta.Status, meta.Project)
			if meta.Completed != "" {
				fmt.Printf(" completed=%s", meta.Completed)
			}
			fmt.Println()
		} else {
			if err := plan.InsertFrontmatter(p.ReadmePath(), meta); err != nil {
				fmt.Printf("  FAIL  P%03d %s: %v\n", p.Number, p.Slug, err)
				failed++
				continue
			}
			fmt.Printf("  OK    P%03d %s\n", p.Number, p.Slug)
		}
		migrated++
	}

	fmt.Printf("\n총 %d개 플랜: %d개 마이그레이션, %d개 스킵(이미 존재), %d개 실패\n",
		len(plans), migrated, skipped, failed)

	if dryRun {
		fmt.Println("\n--dry-run 모드: 파일 변경 없음")
	}

	return nil
}
