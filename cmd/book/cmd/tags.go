package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	tagsPrefix string
	tagsPlan   int
)

// tagsCmd represents the tags command.
//
// Korean: tags 커맨드를 나타낸다.
var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags with usage counts",
	Long: `Query the tag vocabulary from the SQLite index.
Supports filtering by prefix and plan number.

Korean: SQLite 인덱스에서 태그 어휘를 조회한다.
접두사 및 플랜 번호로 필터링을 지원한다.`,
	RunE: runTags,
}

func init() {
	rootCmd.AddCommand(tagsCmd)
	tagsCmd.Flags().StringVar(&tagsPrefix, "prefix", "", "filter tags by prefix (e.g. type:, area:)")
	tagsCmd.Flags().IntVar(&tagsPlan, "plan", 0, "show tags for a specific plan number")
}

func runTags(cmd *cobra.Command, args []string) error {
	s, err := openStoreWithAutoIndex()
	if err != nil {
		return err
	}
	defer s.Close()

	// plan-specific tags
	if tagsPlan > 0 {
		tags, err := s.TagsForPlan(tagsPlan)
		if err != nil {
			return err
		}
		if len(tags) == 0 {
			fmt.Printf("P%03d: 태그 없음\n", tagsPlan)
			return nil
		}
		fmt.Printf("P%03d 태그: [%s]\n", tagsPlan, strings.Join(tags, ", "))
		return nil
	}

	// all tags with counts
	tagCounts, err := s.TagCounts(tagsPrefix)
	if err != nil {
		return err
	}

	if len(tagCounts) == 0 {
		if tagsPrefix != "" {
			fmt.Printf("접두사 '%s'에 해당하는 태그가 없습니다\n", tagsPrefix)
		} else {
			fmt.Println("태그가 없습니다. book index --full을 실행하세요.")
		}
		return nil
	}

	fmt.Printf("%-30s %s\n", "태그", "사용 횟수")
	fmt.Println(strings.Repeat("-", 40))

	for _, tc := range tagCounts {
		fmt.Printf("%-30s %d\n", tc.Tag, tc.Count)
	}

	fmt.Printf("\n총 %d개 태그\n", len(tagCounts))
	return nil
}
