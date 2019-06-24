package history

import (
	"github.com/kitt1987/docker-papa/pkg/home"
	"reflect"
	"strings"
	"time"
)

type log struct {
	Time    time.Time `yaml:"time"`
	Key     string    `yaml:"key"`
	Content string    `yaml:"content"`
}

type logSeries struct {
	Logs []log `yaml:"logs"`
}

type logFile struct {
	path   string
	series logSeries
}

func (f *logFile) Log(key string, content string) {
	if len(f.series.Logs) > 0 {
		lastLog := f.series.Logs[len(f.series.Logs)-1]
		if lastLog.Key == key && reflect.DeepEqual(lastLog.Content, content) {
			f.series.Logs[len(f.series.Logs)-1].Time = time.Now()
			return
		}
	}

	f.series.Logs = append(f.series.Logs, log{
		Time:    time.Now(),
		Key:     key,
		Content: content,
	})

	return
}

func (f *logFile) Search(key string) (content []string) {
	lowerKey := strings.ToLower(key)
	for _, log := range f.series.Logs {
		if strings.ToLower(log.Key) == lowerKey {
			content = append(content, log.Content)
		}
	}

	return
}

func (f *logFile) Truncate(size int) {
	if len(f.series.Logs) > size {
		f.series.Logs = f.series.Logs[size:]
	}

	return
}

func (f *logFile) Close() {
	home.Load().WriteYaml(f.path, f.series)
}

func OpenFile(filePath string) (f File, err error) {
	lf := &logFile{
		path: filePath,
	}

	return lf, home.Load().ReadYaml(filePath, lf.series)
}
