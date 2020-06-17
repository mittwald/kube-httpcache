package watcher

import (
	"github.com/golang/glog"
	"io/ioutil"
	"os"
	"time"
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

	for {
		time.Sleep(15 * time.Second)

		stat, err := os.Stat(t.filename)
		if err != nil {
			errors <- err
			continue
		}

		if stat.ModTime() != t.lastObservedTimestamp {
			glog.V(6).Infof("observed new modification time on %s", t.filename)

			t.lastObservedTimestamp = stat.ModTime()

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
