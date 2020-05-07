package varnishclient

import (
	"github.com/golang/glog"
	"strconv"
)

func (c *client) writeBytes(in []byte) (int, error) {
	glog.V(8).Infof("writing %d bytes to server: %x (%s)", len(in), in, strconv.Quote(string(in)))
	return c.writer.Write(in)
}

func (c *client) writeString(in string) (int, error) {
	return c.writeBytes([]byte(in))
}

func (c *client) SendRequest(method string, args ...string) (*Response, error) {
	return c.sendRequest(method, args...)
}

func (c *client) sendRequest(method string, args ...string) (*Response, error) {
	cmd := method

	glog.V(8).Infof("writing to server: %s", cmd)
	if len(args) > 0 {
		_, err := c.writeString(method + " ")
		if err != nil {
			return nil, err
		}

		for i := range args {
			_, err := c.writeString(args[i])
			if err != nil {
				return nil, err
			}

			if i < len(args)-1 {
				_, err := c.writeBytes([]byte{' '})
				if err != nil {
					return nil, err
				}
			}
		}

		_, err = c.writeBytes([]byte{0x0a})
		if err != nil {
			return nil, err
		}
	} else {
		_, err := c.writeString(method + "\n")
		if err != nil {
			return nil, err
		}
	}

	glog.V(8).Infof("request written; waiting for response")

	return c.readResponse()
}
