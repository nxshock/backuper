package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tidwall/match"
)

/*func winPathToRelative(s string) string {
	ss := strings.Split(s, string(os.PathSeparator))
	return filepath.Join(ss[1:]...)
}*/

func sizeToApproxHuman(s int64) string {
	t := []struct {
		Name string
		Val  int64
	}{
		{"EiB", 1 << 60},
		{"PiB", 1 << 50},
		{"TiB", 1 << 40},
		{"GiB", 1 << 30},
		{"MiB", 1 << 20},
		{"KiB", 1 << 10}}

	for i := 0; i < len(t); i++ {
		v := float64(s) / float64(t[i].Val)
		if v < 1.0 {
			continue
		}

		return fmt.Sprintf("%.1f %s", v, t[i].Name)
	}

	return fmt.Sprintf("%d B", s)
}

// clean убирает невозможнын комбинации символов из пути
func clean(s string) string {
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, `\\`, `\`)
	s = strings.ReplaceAll(s, `//`, `/`)

	return s
}

// stringIn - аналог оператора in
func stringIn(s string, ss []string) (bool, int) {
	for i, v := range ss {
		if v == s {
			return true, i
		}
	}

	return false, -1
}

func isFileNameMatchPatterns(patterns []string, fileName string) bool {
	for _, mask := range patterns {
		if match.Match(filepath.Base(fileName), mask) {
			return true
		}
	}

	return false
}

func isFilePathMatchPatterns(patterns []string, fileName string) bool {
	for _, mask := range patterns {
		if match.Match(fileName, mask) {
			return true
		}
	}

	return false
}
