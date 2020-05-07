package varnishclient

import (
	"fmt"
	"strconv"
	"strings"
)

func (c *client) ListBackends(pattern string) (BackendsResponse, error) {
	var args []string

	if pattern != "" {
		args = []string{"-p", strconv.Quote(pattern)}
	}

	resp, err := c.sendRequest("backends.list", args...)
	if err != nil {
		return nil, err
	}

	if resp.Code != ResponseOK {
		return nil, fmt.Errorf("could not list backends (code %d): %s", resp.Code, string(resp.Body))
	}

	lines := strings.Split(string(resp.Body), "\n")[1:]
	backends := make(BackendsResponse, len(lines))

	for i := range lines {
		name := strings.Split(lines[i], " ")[0]
		backends[i].Name = name
	}

	return backends, nil
}
