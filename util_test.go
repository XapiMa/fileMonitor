package filemonitor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/pkg/errors"
)

func TestParseEvents(t *testing.T) {
	type data struct {
		input    string
		expected int
	}
	tests := []data{
		data{"", 0},
		data{"create", createFlag},
		data{"create|delete", createFlag | deleteFlag},
		data{"rename|write|permission", renameFlag | writeFlag | permissionFlag},
		data{"create|delete|rename|write|permission", createFlag | deleteFlag | renameFlag | writeFlag | permissionFlag},
	}
	for i, test := range tests {
		result, err := parseEvents(test.input)
		if err != nil {
			t.Errorf("[ParseEvents] case %d err: %q", i, err)
		}
		if result != test.expected {
			t.Errorf("[ParseEvents] case %d failed: %d found but expected %d", i, result, test.expected)
		}
	}
}

// instet of addRecursive
func callRecursive(name string, depth, maxdepth int, targetDirs *[]string, pWg *sync.WaitGroup) {
	defer pWg.Done()
	if maxdepth >= 0 {
		if depth > maxdepth {
			return
		}
	}
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in callRecursive")
	}

	dirname := name
	fmt.Printf("dirname : %s\n", dirname)
	if ok := isDir(dirname); !ok {
		fmt.Printf("%s is not exist\n", dirname)
		return
	}
	// err = watcher.Add(dirname)
	// if err != nil {
	// 	logPrint(errorWrap(err))
	// 	return
	// }
	*targetDirs = append(*targetDirs, dirname)
	fileinfos, err := ioutil.ReadDir(dirname)
	if err != nil {
		fmt.Print(errorWrap(err))
		return
	}
	wg := &sync.WaitGroup{}
	for _, fi := range fileinfos {
		wg.Add(1)
		go callRecursive(filepath.Join(dirname, fi.Name()), depth+1, maxdepth, targetDirs, wg)
	}
	wg.Wait()
}

func TestCallRecursive(t *testing.T) {
	type in struct {
		dirsPaths []string
		startPath string
		depth     int
	}
	type ex []string
	type data struct {
		input    in
		expected ex
	}
	tests := []data{
		{
			in{[]string{"1/2/3/4", "1/2/a/b/c"}, "1/2", 0},
			ex{"1"},
		},
		{
			in{[]string{"1/2/3/4", "1/2/a/b/c"}, "1/2", 1},
			ex{"1", "1/2"},
		},
		{
			in{[]string{"1/2/3/4", "1/2/a/b/c"}, "1/2", 2},
			ex{"1", "1/2", "1/2/3", "1/2/a"},
		},
	}
	for i, test := range tests {
		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			t.Errorf("[CallRecursive] case %d err: %q", i, err)
		}
		for j, dirs := range test.input.dirsPaths {
			if err := os.MkdirAll(filepath.Join(tmpDir, dirs), 0755); err != nil {
				t.Errorf("[CallRecursive] case %d,%d err: %q", i, j, err)
			}
		}
		targetDirs := []string{}
		path := filepath.Join(tmpDir, test.input.startPath)
		dir := filepath.Dir(filepath.Clean(path))
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go callRecursive(dir, 0, test.input.depth, &targetDirs, wg)
		wg.Wait()
		expected := []string{}
		for _, exPath := range test.expected {
			expected = append(expected, filepath.Join(tmpDir, exPath))
		}
		sort.Slice(expected, func(i, j int) bool { return expected[i] < expected[j] })
		sort.Slice(targetDirs, func(i, j int) bool { return targetDirs[i] < targetDirs[j] })
		if len(expected) != len(targetDirs) {
			t.Errorf("[CallRecursive] case %d ,len(targetDirs) is %d but expected %d", i, len(targetDirs), len(expected))
			t.Error(targetDirs)
		}
		for j := range targetDirs {
			if targetDirs[j] != expected[j] {
				t.Errorf("[CallRecursive] case %d ,%dth target is %q but expected %q", i, j, targetDirs[j], expected[j])
			}
		}
		os.RemoveAll(tmpDir)
	}
}

// func TestCheckTarget(t *testing.T) {
// 	type in struct {
// 		dMap      map[string]int
// 		check []ta

// 	}
// 	type ta struct{
// 		target string
// 		event int
// 	}
// 	type ex []bool
// 	type data struct {
// 		input    in
// 		expected ex
// 	}
// 	tests := []data{
// 		{
// 			in{
// 				map[string]int{"/1": 0},
// 				[]check{ta{"/",, "/1", "/1/2"},
// 			},
// 			ex{false, true, false},
// 		},
// 		{
// 			in{
// 				map[string]int{"/1/2": 1},
// 				[]string{"/1", "/1/2", "/1/2/3", "/1/2/3/4"},
// 			},
// 			ex{false, true, true, false},
// 		},
// 		{
// 			in{
// 				map[string]int{"/1": -1, "/1/2/3": 0},
// 				[]string{"/", "/1", "/1/2", "/1/2/34/5/6/7/8/9/10"},
// 			},
// 			ex{false, true, true, true},
// 		},
// 	}
// 	for i, test := range tests {
// 		monitor, err := NewMonitor()
// 		if err != nil {
// 			t.Errorf("[CheckTarget] case %d : %s", i, err)
// 		}
// 		monitor.depthMap = test.input.dMap
// 		for j, path := range test.input.checkPath {
// 			ok := monitor.checkTarget(path)
// 			if ok != test.expected[j] {
// 				t.Errorf("[CheckTarget] case %d ,%dth path's ok is %v but expected %v", i, j, ok, test.expected[j])
// 			}
// 		}
// 	}

// }
