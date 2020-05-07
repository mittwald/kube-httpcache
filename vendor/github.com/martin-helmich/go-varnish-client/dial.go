package varnishclient

import (
	"bufio"
	"net"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

// DialTCP connects to an existing Varnish administration port.
// This method does not perform authentication. Use the `Authenticate()` method for that.
func DialTCP(address string) (Client, error) {
	glog.V(7).Infof("connecting to Varnish admin port at %s", address)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)

	client := client{
		reader: reader,
		writer: conn,
	}

	resp, err := client.readResponse()
	if err != nil {
		return nil, err
	}

	if resp.Code == ResponseAuthenticationRequired {
		client.authChallenge = []byte(strings.Split(string(resp.Body), "\n")[0])
		client.authenticationRequired = true

		glog.Infof("authentication required. challenge string: %s", strconv.Quote(string(client.authChallenge)))
	}

	return &client, nil
}
