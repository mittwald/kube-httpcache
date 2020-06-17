package watcher

import (
	"github.com/golang/glog"
	"io/ioutil"
)

func (t *fsnotifyTemplateWatcher) Run() (chan []byte, chan error) {
	updates := make(chan []byte)
	errors := make(chan error)

	go t.watch(updates, errors)

	return updates, errors
}

func (t *fsnotifyTemplateWatcher) watch(updates chan []byte, errors chan error) {
	for ev := range t.watcher.Events {
		glog.V(6).Infof("observed %s event on %s", ev.String(), ev.Name)

		content, err := ioutil.ReadFile(t.filename)
		if err != nil {
			glog.Warningf("error while reading file %s: %s", t.filename, err.Error())

			errors <- err
			continue
		}

		updates <- content
	}
}
