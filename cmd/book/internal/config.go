// Package internal provides shared configuration and utilities for the book CLI.
//
// Korean: book CLI의 공유 설정과 유틸리티를 제공한다.
package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

var overrideRoot string

const defaultBookRoot = "/Users/minkihan/Documents/coda/2026-01-book"

// SetBookRoot overrides the auto-detected book root path.
//
// Korean: 자동 탐색된 book root 경로를 덮어쓴다.
func SetBookRoot(path string) {
	overrideRoot = path
}

// BookRoot returns the book project root directory.
// Resolution order: SetBookRoot > BOOK_ROOT env > parent directory scan > hardcoded fallback.
//
// Korean: book 프로젝트 루트 디렉토리를 반환한다.
// 결정 순서: SetBookRoot > BOOK_ROOT 환경변수 > 상위 디렉토리 탐색 > 하드코딩 폴백.
func BookRoot() (string, error) {
	if overrideRoot != "" {
		return overrideRoot, nil
	}

	// walk up from cwd looking for plans/_templates/readme.template.md
	dir, err := os.Getwd()
	if err != nil {
		return defaultBookRoot, nil
	}

	for {
		marker := filepath.Join(dir, "plans", "_templates", "readme.template.md")
		if _, err := os.Stat(marker); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// fallback: check default exists
	if _, err := os.Stat(filepath.Join(defaultBookRoot, "plans", "_templates", "readme.template.md")); err == nil {
		return defaultBookRoot, nil
	}

	return "", fmt.Errorf("book root를 찾을 수 없음: --book-root 플래그 또는 BOOK_ROOT 환경변수 설정 필요")
}
