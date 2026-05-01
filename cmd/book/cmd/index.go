package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"book/internal"
	"book/internal/store"
)

var fullRebuild bool

// indexCmd represents the index command.
//
// Korean: index 커맨드를 나타낸다.
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Build or update the SQLite index",
	Long: `Build or incrementally update the plan index in .book.db.
Use --full to rebuild the entire index from scratch.

Korean: .book.db의 플랜 인덱스를 구축하거나 증분 업데이트한다.
--full 옵션으로 전체 인덱스를 처음부터 재구축한다.`,
	RunE: runIndex,
}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.Flags().BoolVar(&fullRebuild, "full", false, "rebuild entire index from scratch")
}

func runIndex(cmd *cobra.Command, args []string) error {
	root, err := internal.BookRoot()
	if err != nil {
		return err
	}

	s, err := store.Open(root)
	if err != nil {
		return err
	}
	defer s.Close()

	start := time.Now()

	if fullRebuild {
		fmt.Println("전체 인덱스 재구축 중...")
		if err := s.RebuildAll(); err != nil {
			return err
		}
	} else {
		fmt.Println("변경된 플랜 인덱싱 중...")
		if err := s.UpdateChanged(); err != nil {
			return err
		}
	}

	fmt.Printf("완료 (%s)\n", time.Since(start).Round(time.Millisecond))
	return nil
}
