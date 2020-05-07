package varnishclient

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func (c *client) Authenticate(secret []byte) error {
	input := string(c.authChallenge) + "\n" + string(secret) + string(c.authChallenge) + "\n"
	response := sha256.Sum256([]byte(input))
	responseHex := hex.EncodeToString(response[:])

	resp, err := c.sendRequest("auth", responseHex)
	if err != nil {
		return err
	}

	if resp.Code != ResponseOK {
		return fmt.Errorf("response code was %d, expected %d", resp.Code, ResponseOK)
	}

	c.authenticated = true

	return nil
}

func (c *client) AuthenticationRequired() bool {
	return c.authenticationRequired
}
