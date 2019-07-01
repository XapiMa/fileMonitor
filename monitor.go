package filemonitor

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

// Monitor is monitoring object
type Monitor struct {
	targets    map[string]target
	depthMap   map[string]int
	watcher    *fsnotify.Watcher
	configPath string
	outputPath string
	messages   map[int]string
}

type target struct {
	path   string
	depth  int
	events int
}

// NewMonitor create new monitoring object
func NewMonitor() (*Monitor, error) {
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in NewMonitor")
	}
	m := new(Monitor)
	m.targets = make(map[string]target)
	m.depthMap = make(map[string]int)
	var err error
	m.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return m, errorWrap(err)
	}
	m.configPath = ""
	m.outputPath = ""
	m.messages = map[int]string{
		createFlag:     "CREATE",
		deleteFlag:     "DELETE",
		writeFlag:      "WRITE",
		renameFlag:     "RENAME",
		permissionFlag: "PERNISSION",
	}
	return m, nil
}

// defer watcher.Close()

func (monitor *Monitor) parseConfigFile(configPath string) error {
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in parseConfigFile")
	}

	buf, err := ioutil.ReadFile(configPath)
	if err != nil {
		return errorWrap(err)
	}

	data, err := ymlUnmarshal(buf)
	if err != nil {
		return errorWrap(err)
	}

	for i, item := range data {
		// default: depth == 0 and all event watch
		tmpItem := target{"", 0, createFlag | deleteFlag | renameFlag | writeFlag | permissionFlag}
		if val, ok := item["path"]; ok {
			tmpItem.path = filepath.Clean(val)
		} else {
			return errorWrap(fmt.Errorf("%dth item don't have path in %s", i+1, configPath))
		}
		if val, ok := item["depth"]; ok {
			if tmpItem.depth, err = strconv.Atoi(val); err != nil {
				return errorWrap(fmt.Errorf("%dth item's depth expect int but found %s", i+1, val))
			}
		}
		if val, ok := item["event"]; ok {
			if tmpItem.events, err = parseEvents(val); err != nil {
				return errorWrap(err)
			}

		}
		monitor.targets[tmpItem.path] = tmpItem
		monitor.depthMap[tmpItem.path] = tmpItem.depth
	}
	return nil

}

// FileMonitor is monitoring file system
func (monitor *Monitor) FileMonitor(configPath, outputPath string, maxParallelNum int) error {
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in FileMonitor")
	}
	monitor.configPath = configPath
	err := monitor.parseConfigFile(configPath)
	if err != nil {
		return errorWrap(err)
	}

	monitor.outputPath = outputPath

	done := make(chan bool)
	go monitor.watch(outputPath, done)

	// add all parent dir of target to watcher
	wg := &sync.WaitGroup{}
	for path, item := range monitor.targets {
		dirname := filepath.Dir(path)
		wg.Add(1)
		go monitor.addRecursive(dirname, 0, item.depth, wg)
	}
	wg.Wait()
	<-done

	return nil

}

func (monitor *Monitor) watch(outputPath string, done chan bool) {
	errorWrap := func(err error) error {
		return errors.Wrap(err, "cause in watch")
	}
	prevString := ""
	for {
		eventType := 0
		select {
		case event := <-monitor.watcher.Events:
			timeString := time.Now().Format("2006/01/02 15:04:05")
			// event.Nameをたどってtargetのpath内に含まれているか確認
			// 含まれていた場合，さかのぼった回数がdepthに見合っているか確認
			// 見合っていなければ無視
			// 例) path=/etc/passwd,depth=0 の場合
			// とりあえず親dirがAddされているので，/etc直下のファイルの全情報が返ってくる
			// event.Nameが/etc/passwdの場合，0回さかのぼってtargetsに含まれることが発見でき，これは正しいので処理される
			// ここで，path=/etc, depth=0 もtargetだった場合を考える
			// /etc/lib に変更が加わると ath=/etc/passwd,depth=0 を監視しているために通知される
			// このとき，/etc/lib は1回さかのぼって/etcにマッチする
			// /etcの監視範囲はdepth=0なので無視される
			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				eventType = createFlag
				if monitor.checkTargetPath(event.Name) {
					go monitor.addDir(event.Name)
				}
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				eventType = deleteFlag
			case event.Op&fsnotify.Write == fsnotify.Write:
				eventType = writeFlag
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				eventType = renameFlag
			case event.Op&fsnotify.Chmod == fsnotify.Chmod:
				eventType = permissionFlag
			}
			outputString := fmt.Sprintf("%s %s: %s", timeString, monitor.messages[eventType], event.Name)
			if prevString != outputString {
				if monitor.checkTarget(event.Name, eventType) {
					monitor.appendFile(outputString)
				}
			}
			prevString = outputString

		case err := <-monitor.watcher.Errors:
			logPrint(errorWrap(err))
			done <- true
		}
	}
}

func (monitor *Monitor) addDir(filename string) {
	// event.Nameをさかのぼり，depthが設定されたpathを探す
	// 見つけたら，さかのぼった数がdepthに見合っているか確認する
	// 見合っていれば監視対象に加える
	ok := isDir(filename)
	if ok {
		if ok := monitor.checkTargetPath(filename); ok {
			monitor.watcher.Add(filename)
		}
	}

}

// Close removes all watches and closes the events channel.
func (monitor *Monitor) Close() {
	monitor.watcher.Close()
}
