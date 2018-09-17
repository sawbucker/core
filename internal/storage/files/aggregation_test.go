package files_test

import (
	"testing"
	"time"

	"github.com/ShoshinNikita/tags-drive/internal/storage/files"
)

func TestIsGoodFile(t *testing.T) {
	tests := []struct {
		m     files.TagMode
		fTags []string
		pTags []string
		res   bool
	}{
		{files.ModeAnd, []string{"a", "b", "c"}, []string{"a", "c"}, true},
		{files.ModeAnd, []string{"a", "b", "c"}, []string{"a", "e"}, false},
		{files.ModeOr, []string{"a", "b", "c"}, []string{"a", "e"}, true},
		{files.ModeOr, []string{"a", "b", "c"}, []string{"f", "e"}, false},
		{files.ModeNot, []string{"p", "b", "c"}, []string{"a", "e"}, true},
		{files.ModeNot, []string{"a", "b", "c"}, []string{"a", "e"}, false},
		// Empty file tags
		{files.ModeAnd, []string{}, []string{"a", "e"}, false},
		{files.ModeOr, []string{}, []string{"a", "e"}, false},
		{files.ModeNot, []string{}, []string{"a", "e"}, true},
		// Empty passed tags
		{files.ModeAnd, []string{"a", "b", "c"}, []string{}, true},
		{files.ModeOr, []string{"a", "b", "c"}, []string{}, true},
		{files.ModeNot, []string{"a", "b", "c"}, []string{}, true},
	}

	for i, tt := range tests {
		res := files.IsGoodFile(tt.m, tt.fTags, tt.pTags)
		if res != tt.res {
			t.Errorf("Test #%d Want: %v Got %v", i, tt.res, res)
		}
	}

}

func TestSortFiles(t *testing.T) {
	getTime := func(s string) time.Time {
		tm, err := time.Parse("01-02-2006 15:04:05", s)
		if err != nil {
			t.Errorf("Bad time %s", s)
		}
		return tm
	}

	isEqual := func(a, b []files.FileInfo) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i].Filename != b[i].Filename ||
				a[i].AddTime != b[i].AddTime ||
				a[i].Size != b[i].Size {
				return false
			}
		}

		return true
	}

	tests := []struct {
		s     files.SortMode
		files []files.FileInfo
		res   []files.FileInfo
	}{
		{files.SortByNameAsc,
			[]files.FileInfo{
				files.FileInfo{Filename: "abc"},
				files.FileInfo{Filename: "cbd"},
				files.FileInfo{Filename: "aaa"},
				files.FileInfo{Filename: "fer"},
			},
			[]files.FileInfo{
				files.FileInfo{Filename: "aaa"},
				files.FileInfo{Filename: "abc"},
				files.FileInfo{Filename: "cbd"},
				files.FileInfo{Filename: "fer"},
			},
		},
		{files.SortByNameDesc,
			[]files.FileInfo{
				files.FileInfo{Filename: "abc"},
				files.FileInfo{Filename: "cbd"},
				files.FileInfo{Filename: "aaa"},
				files.FileInfo{Filename: "fer"},
			},
			[]files.FileInfo{
				files.FileInfo{Filename: "fer"},
				files.FileInfo{Filename: "cbd"},
				files.FileInfo{Filename: "abc"},
				files.FileInfo{Filename: "aaa"},
			},
		},
		{files.SortByTimeAsc,
			[]files.FileInfo{
				files.FileInfo{AddTime: getTime("05-05-2018 15:45:35")},
				files.FileInfo{AddTime: getTime("05-05-2018 15:22:35")},
				files.FileInfo{AddTime: getTime("05-05-2018 15:16:35")},
				files.FileInfo{AddTime: getTime("05-04-2018 15:22:35")},
			},
			[]files.FileInfo{
				files.FileInfo{AddTime: getTime("05-04-2018 15:22:35")},
				files.FileInfo{AddTime: getTime("05-05-2018 15:16:35")},
				files.FileInfo{AddTime: getTime("05-05-2018 15:22:35")},
				files.FileInfo{AddTime: getTime("05-05-2018 15:45:35")},
			},
		},
		{files.SortByTimeDesc,
			[]files.FileInfo{
				files.FileInfo{AddTime: getTime("05-05-2018 15:45:35")},
				files.FileInfo{AddTime: getTime("05-05-2018 15:22:35")},
				files.FileInfo{AddTime: getTime("05-05-2018 15:16:35")},
				files.FileInfo{AddTime: getTime("05-04-2018 15:22:35")},
			},
			[]files.FileInfo{
				files.FileInfo{AddTime: getTime("05-05-2018 15:45:35")},
				files.FileInfo{AddTime: getTime("05-05-2018 15:22:35")},
				files.FileInfo{AddTime: getTime("05-05-2018 15:16:35")},
				files.FileInfo{AddTime: getTime("05-04-2018 15:22:35")},
			},
		},
		{files.SortBySizeAsc,
			[]files.FileInfo{
				files.FileInfo{Size: 15},
				files.FileInfo{Size: 1515},
				files.FileInfo{Size: 1885},
				files.FileInfo{Size: 1365},
				files.FileInfo{Size: 1551561651},
			},
			[]files.FileInfo{
				files.FileInfo{Size: 15},
				files.FileInfo{Size: 1365},
				files.FileInfo{Size: 1515},
				files.FileInfo{Size: 1885},
				files.FileInfo{Size: 1551561651},
			},
		},
		{files.SortBySizeDecs,
			[]files.FileInfo{
				files.FileInfo{Size: 15},
				files.FileInfo{Size: 1515},
				files.FileInfo{Size: 1885},
				files.FileInfo{Size: 1365},
				files.FileInfo{Size: 1551561651},
			},
			[]files.FileInfo{
				files.FileInfo{Size: 1551561651},
				files.FileInfo{Size: 1885},
				files.FileInfo{Size: 1515},
				files.FileInfo{Size: 1365},
				files.FileInfo{Size: 15},
			},
		},
	}

	for i, tt := range tests {
		files.SortFiles(tt.s, tt.files)
		if !isEqual(tt.files, tt.res) {
			t.Errorf("Test #%d Want: %v Got: %v", i, tt.res, tt.files)
		}
	}
}