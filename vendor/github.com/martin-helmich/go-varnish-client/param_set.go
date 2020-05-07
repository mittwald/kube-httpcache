package varnishclient

import (
	"fmt"
	"strconv"
)

func (c *client) SetParameter(name, value string) error {
	resp, err := c.sendRequest("param.set", name, strconv.Quote(value))
	if err != nil {
		return err
	}

	if resp.Code != ResponseOK {
		return fmt.Errorf("could not set parameter '%s' (code %d): %s", name, resp.Code, string(resp.Body))
	}

	return nil
}