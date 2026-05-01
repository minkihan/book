package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"book/internal"
	"book/internal/plan"
	"book/internal/store"
)

// noteCmd represents the note command (add a note).
//
// Korean: note 커맨드를 나타낸다 (노트 추가).
var noteCmd = &cobra.Command{
	Use:   "note [plan-number] [text]",
	Short: "Add a note to a plan",
	Long: `Add a timestamped note entry to a plan's notes.md file.
The note is also synced to the SQLite index.

Korean: 플랜의 notes.md 파일에 타임스탬프가 포함된 노트를 추가한다.
SQLite 인덱스에도 동기화된다.`,
	Args: cobra.MinimumNArgs(2),
	RunE: runNote,
}

func init() {
	rootCmd.AddCommand(noteCmd)
}

func runNote(cmd *cobra.Command, args []string) error {
	num, err := parseNumberArg(args[0])
	if err != nil {
		return err
	}

	text := strings.Join(args[1:], " ")

	root, err := internal.BookRoot()
	if err != nil {
		return err
	}

	p, err := plan.FindPlan(root, num)
	if err != nil {
		return err
	}

	if err := plan.AppendNote(p.NotesPath(), num, text); err != nil {
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

	fmt.Printf("P%03d에 노트 추가 완료\n", num)
	return nil
}
