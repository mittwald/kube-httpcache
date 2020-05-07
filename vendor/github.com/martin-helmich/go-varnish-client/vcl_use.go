package varnishclient

import (
	"fmt"
	"strconv"
	"strings"
)

func (c *client) UseVCL(configname string) error {
	resp, err := c.sendRequest("vcl.use", strconv.Quote(configname))
	if err != nil {
		return err
	}

	if resp.Code != ResponseOK {
		return fmt.Errorf("error while activating VCL (code %d): %s", resp.Code, strings.TrimSpace(string(resp.Body)))
	}

	return nil
}
