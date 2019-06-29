package filemonitor

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

type target struct {
	path   string
	depth  int
	events int
}

var depthMap = make(map[string]int)

// FileMonitor is monitoring file system
func FileMonitor(configPath, outputPath string, maxParallelNum int) error {
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in FileMonitor")
	}

	targets, err := parseConfigFile(configPath)
	if err != nil {
		return errorWrap(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errorWrap(err)
	}
	defer watcher.Close()
	done := make(chan bool)

	go watch(outputPath, watcher, done)

	// add all target parent dir to watcher
	wg := &sync.WaitGroup{}
	for _, item := range targets {
		dirname := filepath.Dir(item.path)
		wg.Add(1)
		go addRecursive(dirname, 0, item.depth, watcher, wg)
	}
	wg.Wait()

	<-done

	return nil

}

func parseConfigFile(configPath string) ([]target, error) {
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in parseConfigFile")
	}
	var targets = []target{}

	buf, err := ioutil.ReadFile(configPath)
	if err != nil {
		return targets, errorWrap(err)
	}

	data, err := readOnSliceMap(buf)
	if err != nil {
		return targets, errorWrap(err)
	}

	for i, item := range data {
		// default: depth == 0 and all event watch
		tmpItem := target{"", 0, createFlag | removeFlag | renameFlag | writeFlag | permissionFlag}
		if val, ok := item["path"]; ok {
			tmpItem.path = filepath.Clean(val)
		} else {
			return targets, errorWrap(fmt.Errorf("%dth item don't have path in %s", i+1, configPath))
		}
		if val, ok := item["depth"]; ok {
			tmpItem.depth, err = strconv.Atoi(val)
			if err != nil {
				return targets, errorWrap(fmt.Errorf("%dth item's depth expect int but found %s", i+1, val))
			}
		}
		if val, ok := item["events"]; !ok {
			tmpItem.events, err = parseEvents(val)
			if err != nil {
				return targets, errorWrap(err)
			}
		}
		targets = append(targets, tmpItem)
		depthMap[tmpItem.path] = tmpItem.depth
	}
	return targets, nil
}

func watch(outputPath string, watcher *fsnotify.Watcher, done chan bool) {
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in watch")
	}
	for {
		select {
		case event := <-watcher.Events:
			// event.Nameをたどったうちの何処かがpath内に含まれているか確認
			// 含まれていた場合，さかのぼった回数がdepthに見合っているか確認
			// 見合っていなければ無視
			// 例) path=/etc/passwd,depth=0 の場合
			// とりあえず親dirがAddされているので，/etc直下のファイルの全情報が返ってくる
			// event.Nameが/etc/passwdの場合，0回さかのぼってtargetsに含まれることが発見でき，これは正しいので処理される
			// ここで，path=/etc, depth=0 もtargetだった場合を考える
			// /etc/lib に変更が加わると ath=/etc/passwd,depth=0 を監視しているために通知される
			// このとき，/etc/lib は1回さかのぼって/etcにマッチする
			// /etcの監視範囲はdepth=0なので無視される
			if ok := checkTarget(event.Name); !ok {
				continue
			}
			outputString := ""
			switch {
			case event.Op&fsnotify.Write == fsnotify.Write:
				outputString += "Modified : " + event.Name
			case event.Op&fsnotify.Create == fsnotify.Create:
				outputString += "Created: " + event.Name
				// event.Nameをさかのぼり，depthが設定されたpathを探す
				// 見つけたら，さかのぼった数がdepthに見合っているか確認する
				// 見合っていれば監視対象に加える
				ok, err := isDirExist(event.Name)
				if err != nil {
					log.Println("error: ", err)
				}
				if ok {
					if ok := checkTarget(event.Name); ok {
						watcher.Add(event.Name)
					}
				}
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				outputString += "Removed: " + event.Name
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				outputString += "Renamed: " + event.Name
			case event.Op&fsnotify.Chmod == fsnotify.Chmod:
				outputString += "Permission: " + event.Name
			}
			if err := appendFile(outputPath, outputString+"\n"); err != nil {
				logPrint(errorWrap(err))
			}

		case err := <-watcher.Errors:
			logPrint(errorWrap(err))
			done <- true
		}
	}

}
