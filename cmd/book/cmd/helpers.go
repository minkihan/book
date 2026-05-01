package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"book/internal"
	"book/internal/store"
)

// openStoreWithAutoIndex opens the store and runs incremental indexing.
//
// Korean: 스토어를 열고 증분 인덱싱을 실행한다.
func openStoreWithAutoIndex() (*store.Store, error) {
	root, err := internal.BookRoot()
	if err != nil {
		return nil, err
	}
	s, err := store.Open(root)
	if err != nil {
		return nil, err
	}
	// auto incremental indexing on query commands
	_ = s.UpdateChanged()
	return s, nil
}

// parseNumberArg parses a plan number string → (number, error).
//
// Korean: 플랜 번호 문자열을 파싱하여 (number, error)를 반환한다.
func parseNumberArg(s string) (int, error) {
	if strings.Contains(s, ":") {
		return 0, fmt.Errorf("'%s': year:number 형식은 더 이상 지원되지 않습니다. 글로벌 번호만 사용하세요", s)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("유효하지 않은 플랜 번호: %s", s)
	}
	return n, nil
}
