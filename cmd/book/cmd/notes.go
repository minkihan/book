package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"book/internal"
	"book/internal/plan"
)

var lastN int

// notesCmd represents the notes command (view notes).
//
// Korean: notes 커맨드를 나타낸다 (노트 조회).
var notesCmd = &cobra.Command{
	Use:   "notes [plan-number]",
	Short: "View notes for a plan",
	Long: `View all notes for a plan in reverse chronological order.
Use --last N to show only the N most recent notes.

Korean: 플랜의 모든 노트를 최신순으로 조회한다.
--last N으로 최근 N개만 표시한다.`,
	Args: cobra.ExactArgs(1),
	RunE: runNotes,
}

func init() {
	rootCmd.AddCommand(notesCmd)
	notesCmd.Flags().IntVar(&lastN, "last", 0, "show only last N notes")
}

func runNotes(cmd *cobra.Command, args []string) error {
	num, err := parseNumberArg(args[0])
	if err != nil {
		return err
	}

	root, err := internal.BookRoot()
	if err != nil {
		return err
	}

	p, err := plan.FindPlan(root, num)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(p.NotesPath())
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("P%03d: 노트 없음\n", num)
			return nil
		}
		return err
	}

	notes, err := plan.ParseNotes(data)
	if err != nil {
		return err
	}

	if len(notes) == 0 {
		fmt.Printf("P%03d: 노트 없음\n", num)
		return nil
	}

	max := len(notes)
	if lastN > 0 && lastN < max {
		max = lastN
	}

	fmt.Printf("P%03d 노트 (%d개)\n\n", num, len(notes))
	for i := 0; i < max; i++ {
		fmt.Printf("[%s]\n%s\n\n", notes[i].Timestamp, notes[i].Content)
	}

	return nil
}
