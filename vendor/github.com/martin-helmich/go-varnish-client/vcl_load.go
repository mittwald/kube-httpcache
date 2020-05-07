package varnishclient

import (
	"fmt"
	"strconv"
)

func (c *client) LoadVCL(configname, filename string, mode string) error {
	resp, err := c.sendRequest("vcl.load", strconv.Quote(configname), strconv.Quote(filename), mode)
	if err != nil {
		return err
	}

	if resp.Code != ResponseOK {
		return fmt.Errorf("error while loading VCL (code %d): %s", resp.Code, string(resp.Body))
	}

	return nil
}