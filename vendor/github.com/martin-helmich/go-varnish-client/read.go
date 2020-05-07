package varnishclient

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

func (c *client) readResponse() (*Response, error) {
	header := make([]byte, 13)

	n, err := io.ReadFull(c.reader, header)
	if err != nil {
		return nil, err
	}

	glog.V(8).Infof("read %d bytes of header", n)
	glog.V(8).Infof("header: %s", strconv.Quote(string(header)))

	code, err := strconv.Atoi(string(header[0:3]))
	if err != nil {
		return nil, err
	}

	blen, err := strconv.Atoi(strings.TrimSpace(string(header[3:11])))
	if err != nil {
		return nil, fmt.Errorf("invalid length: %s", err.Error())
	}

	glog.V(8).Infof("received message from Varnish server: response code %d, body length %d", code, blen)

	body := make([]byte, blen+1)
	m, err := io.ReadFull(c.reader, body)

	if m != blen+1 {
		return nil, fmt.Errorf("incomplete body: only %d bytes read, %d expected", m, blen)
	}

	glog.V(8).Infof("%d bytes read", m)
	glog.V(8).Infof("message body: %s", strconv.Quote(string(body)))

	response := Response{}
	response.Code = code
	response.Body = body

	return &response, nil
}
