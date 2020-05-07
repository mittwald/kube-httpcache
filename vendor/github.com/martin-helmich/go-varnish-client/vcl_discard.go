package varnishclient

import (
	"fmt"
	"strconv"
	"strings"
)

func (c *client) DiscardVCL(configname string) error {
	resp, err := c.sendRequest("vcl.discard", strconv.Quote(configname))
	if err != nil {
		return err
	}

	if resp.Code != ResponseOK {
		return fmt.Errorf("error while discarding VCL (code %d): %s", resp.Code, strings.TrimSpace(string(resp.Body)))
	}

	return nil
}
