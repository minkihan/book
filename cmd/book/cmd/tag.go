package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"book/internal"
	"book/internal/plan"
	"book/internal/store"
)

// tagCmd represents the tag command.
//
// Korean: tag 커맨드를 나타낸다.
var tagCmd = &cobra.Command{
	Use:   "tag [plan-number] [add|remove] [tag]",
	Short: "Add or remove tags from a plan",
	Long: `Modify tags in a plan's frontmatter and sync to SQLite.

Korean: 플랜의 frontmatter 태그를 수정하고 SQLite에 동기화한다.`,
	Args: cobra.ExactArgs(3),
	RunE: runTag,
}

func init() {
	rootCmd.AddCommand(tagCmd)
}

func runTag(cmd *cobra.Command, args []string) error {
	num, err := parseNumberArg(args[0])
	if err != nil {
		return err
	}

	action := args[1]
	tag := args[2]

	if action != "add" && action != "remove" {
		return fmt.Errorf("유효하지 않은 액션: %s (add 또는 remove)", action)
	}

	root, err := internal.BookRoot()
	if err != nil {
		return err
	}

	p, err := plan.FindPlan(root, num)
	if err != nil {
		return err
	}

	result, err := plan.ParseFrontmatter(p.ReadmePath())
	if err != nil {
		return err
	}

	meta := result.Meta
	if meta == nil {
		return fmt.Errorf("P%03d: frontmatter가 없습니다. 먼저 book migrate를 실행하세요", num)
	}

	switch action {
	case "add":
		// check for duplicate
		for _, t := range meta.Tags {
			if t == tag {
				fmt.Printf("P%03d: 태그 '%s'가 이미 존재합니다\n", num, tag)
				return nil
			}
		}
		meta.Tags = append(meta.Tags, tag)
	case "remove":
		found := false
		filtered := make([]string, 0, len(meta.Tags))
		for _, t := range meta.Tags {
			if t == tag {
				found = true
				continue
			}
			filtered = append(filtered, t)
		}
		if !found {
			fmt.Printf("P%03d: 태그 '%s'를 찾을 수 없습니다\n", num, tag)
			return nil
		}
		meta.Tags = filtered
	}

	if err := plan.WriteFrontmatter(p.ReadmePath(), meta); err != nil {
		return err
	}

	// sync to SQLite
	s, err := store.Open(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: SQLite 동기화 실패: %v\n", err)
	} else {
		if syncErr := s.IndexPlan(num); syncErr != nil {
			fmt.Fprintf(os.Stderr, "WARN: 인덱스 업데이트 실패: %v\n", syncErr)
		}
		s.Close()
	}

	fmt.Printf("P%03d: 태그 %s '%s' 완료\n", num, action, tag)
	return nil
}
