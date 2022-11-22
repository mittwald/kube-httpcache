package watcher

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/golang/glog"
	"io/ioutil"
	"os"
	"time"
)

const (
	// how often to check for template file changes
	POLL_INTERVAL = 15 * time.Second
	// how often print template info for troubleshooting
	TIMESTAMP_DISPLAY_INTERVAL = 1 * time.Hour
)

func (t *pollingTemplateWatcher) Run() (chan []byte, chan error) {
	updates := make(chan []byte)
	errors := make(chan error)

	go t.watch(updates, errors)

	return updates, errors
}

func (t *pollingTemplateWatcher) watch(updates chan []byte, errors chan error) {
	stat, err := os.Stat(t.filename)
	if err != nil {
		errors <- err
	}

	t.lastObservedTimestamp = stat.ModTime()
	glog.V(6).Infof("observed modification time on %s (%s)", t.filename, t.lastObservedTimestamp.String())

	var i uint64 = 0
	logTemplateInfoCount := uint64(TIMESTAMP_DISPLAY_INTERVAL / POLL_INTERVAL)
	for {
		time.Sleep(POLL_INTERVAL)

		stat, err := os.Stat(t.filename)
		if err != nil {
			errors <- err
			continue
		}

		modtime := stat.ModTime()
		i++
		if glog.V(6) && (i%logTemplateInfoCount == 0) {
			logTemplateInfo(t.filename, modtime, errors)
		}

		if modtime != t.lastObservedTimestamp {
			glog.V(6).Infof("observed new modification time on %s (%s)", t.filename, modtime.String())

			t.lastObservedTimestamp = modtime

			content, err := ioutil.ReadFile(t.filename)
			if err != nil {
				glog.Warningf("error while reading file %s: %s", t.filename, err.Error())

				errors <- err
				continue
			}

			updates <- content
		}
	}
}

// print template info to assist troubleshooting
func logTemplateInfo(filename string, modtime time.Time, errors chan error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		glog.Warningf("error while reading file %s: %s", filename, err.Error())
		errors <- err
		return
	}

	hash := md5.Sum(content)
	hashStr := hex.EncodeToString(hash[:])
	glog.Infof("current template modification time: %s, md5sum: %s", modtime.String(), hashStr)
}
