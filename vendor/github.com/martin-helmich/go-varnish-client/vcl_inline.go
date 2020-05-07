package varnishclient

import (
	"fmt"
	"strconv"
)

const (
	VCLModeAuto = "auto"
	VCLModeCold = "cold"
	VCLModeWarm = "warm"
)

func (c *client) DefineInlineVCL(configname string, vcl []byte, mode string) error {
	resp, err := c.sendRequest("vcl.inline", strconv.Quote(configname), strconv.Quote(string(vcl)), mode)
	if err != nil {
		return err
	}

	if resp.Code != ResponseOK {
		return fmt.Errorf("error while loading VCL (code %d): %s", resp.Code, string(resp.Body))
	}

	return nil
}