package varnishclient

import (
	"bufio"
	"io"
)

// Response contains the data that was received from Varnish in response to a request
type Response struct {
	Code int
	Body []byte
}

// Backend is a single item of the list returned by the `ListBackends` method
type Backend struct {
	Name string
}

// BackendsResponse is the respose type of the `ListBackends` method
type BackendsResponse []Backend

// Parameter is a single item of the list returned by the `ListParameters` method
type Parameter struct {
	Name      string
	Value     string
	Unit      string
	IsDefault bool
}

// ParametersResponse is the response type of the `ListParameters` method
type ParametersResponse []Parameter

type client struct {
	authChallenge []byte
	reader        io.Reader
	writer        io.Writer
	scanner       *bufio.Scanner

	authenticationRequired bool
	authenticated          bool
}

// Client contains the most common Varnish administration operations (and the
// necessary tools to build your own that are not yet implemented)
type Client interface {
	AuthenticationRequired() bool
	Authenticate([]byte) error
	ListBackends(pattern string) (BackendsResponse, error)

	SetParameter(name, value string) error
	ListParameters() (ParametersResponse, error)

	DiscardVCL(configName string) error
	DefineInlineVCL(configName string, vcl []byte, mode string) error
	AddLabelToVCL(label string, configName string) error
	LoadVCL(configName, filename string, mode string) error
	UseVCL(configName string) error
}
