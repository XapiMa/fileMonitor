package filemonitor

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-yaml/yaml"
	"github.com/pkg/errors"
)

func logPrint(err error) {
	log.Printf("Error: webStatusChecker %s %s", time.Now(), err)
}

func parseEvents(eventsString string) (int, error) {
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in parseEvents")
	}
	events := strings.Split(eventsString, "|")
	eventFlag := 0
	for i, event := range events {
		switch event {
		case createSentence:
			eventFlag |= createFlag
		case removeSentence:
			eventFlag |= removeFlag
		case renameSentence:
			eventFlag |= renameFlag
		case writeSentence:
			eventFlag |= writeFlag
		case permissionSentence:
			eventFlag |= permissionFlag
		case "":
		default:
			return eventFlag, errorWrap(fmt.Errorf("%dth item's events is invalit: %s", i, eventsString))
		}
	}
	return eventFlag, nil
}

func isDir(directory string) bool {
	fInfo, err := os.Stat(directory)
	if err == nil {
		if fInfo.IsDir() {
			return true
		}
	}
	return false
}

func appendFile(outputPath, outputString string) error {
	outputString = fmt.Sprintf("%s\n", outputString)
	if outputPath == "" {
		fmt.Printf("%s", outputString)
	} else {
		file, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			logPrint(err)
			return err
		}
		file.Write(([]byte)(outputString))
		file.Close()
	}
	return nil
}

func addRecursive(name string, depth, maxdepth int, watcher *fsnotify.Watcher, pWg *sync.WaitGroup) {
	defer pWg.Done()
	if maxdepth >= 0 {
		if depth > maxdepth {
			return
		}
	}

	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in addRecursive")
	}

	dirname := name
	if ok := isDir(dirname); !ok {
		return
	}

	if err := watcher.Add(dirname); err != nil {
		logPrint(errorWrap(err))
		return
	}

	fileinfos, err := ioutil.ReadDir(dirname)
	if err != nil {
		logPrint(errorWrap(err))
		return
	}
	wg := &sync.WaitGroup{}
	for _, fi := range fileinfos {
		wg.Add(1)
		go addRecursive(filepath.Join(dirname, fi.Name()), depth+1, maxdepth, watcher, wg)
	}

	wg.Wait()

}

func checkTarget(path string) bool {
	for i := 0; true; i++ {
		depth, ok := depthMap[path]
		if ok {
			if depth >= i || depth < 0 {
				return true
			}
			return false
		}
		dir, file := filepath.Split(path)
		dir = filepath.Clean(dir)
		if file == "" {
			return false
		}
		path = dir
	}
	return true
}

func readOnSliceMap(fileBuffer []byte) ([]map[string]string, error) {
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in ReadOnSliceMap")
	}
	data := make([]map[string]string, 1000)
	err := yaml.Unmarshal(fileBuffer, &data)
	if err != nil {
		return nil, errorWrap(err)
	}
	return data, nil
}
