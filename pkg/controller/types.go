package controller

import (
	"github.com/golang/glog"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/mittwald/kube-httpcache/pkg/signaller"
	"github.com/mittwald/kube-httpcache/pkg/watcher"
)

type TemplateData struct {
	Frontends       watcher.EndpointList
	PrimaryFrontend *watcher.Endpoint
	Backends        watcher.EndpointList
	PrimaryBackend  *watcher.Endpoint
	Env             map[string]string
}

type VarnishController struct {
	SecretFile           string
	Storage              string
	TransientStorage     string
	AdditionalParameters string
	WorkingDir           string
	FrontendAddr         string
	FrontendPort         int
	AdminAddr            string
	AdminPort            int

	vclTemplate *template.Template
	// md5 hash of unparsed template
	vclTemplateHash    string
	vclTemplateUpdates chan []byte
	frontendUpdates    chan *watcher.EndpointConfig
	frontend           *watcher.EndpointConfig
	backendUpdates     chan *watcher.EndpointConfig
	backend            *watcher.EndpointConfig
	varnishSignaller   *signaller.Signaller
	configFile         string
	localAdminAddr     string
	currentVCLName     string
}

func NewVarnishController(
	secretFile string,
	storage string,
	transientStorage string,
	additionalParameter string,
	workingDir string,
	frontendAddr string,
	frontendPort int,
	adminAddr string,
	adminPort int,
	frontendUpdates chan *watcher.EndpointConfig,
	backendUpdates chan *watcher.EndpointConfig,
	templateUpdates chan []byte,
	varnishSignaller *signaller.Signaller,
	vclTemplateFile string,
) (*VarnishController, error) {
	contents, err := os.ReadFile(vclTemplateFile)
	if err != nil {
		return nil, err
	}

	v := VarnishController{
		SecretFile:           secretFile,
		Storage:              storage,
		TransientStorage:     transientStorage,
		AdditionalParameters: additionalParameter,
		WorkingDir:           workingDir,
		FrontendAddr:         frontendAddr,
		FrontendPort:         frontendPort,
		AdminAddr:            adminAddr,
		AdminPort:            adminPort,
		vclTemplateUpdates:   templateUpdates,
		frontendUpdates:      frontendUpdates,
		backendUpdates:       backendUpdates,
		varnishSignaller:     varnishSignaller,
		configFile:           "/tmp/vcl",
	}
	err = v.setTemplate(contents)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

func getEnvironment() map[string]string {
	items := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		items[pair[0]] = pair[1]
	}
	return items
}

func (v *VarnishController) renderVCL(target io.Writer, frontendList watcher.EndpointList, primaryFrontend *watcher.Endpoint, backendList watcher.EndpointList, primaryBackend *watcher.Endpoint) error {
	glog.V(6).Infof("rendering VCL (source md5sum: %s, Frontends:%v, PrimaryFrontend:%v, Backends:%v, PrimaryBackend:%v)",
		v.vclTemplateHash, frontendList, primaryFrontend, backendList, primaryBackend)

	err := v.vclTemplate.Execute(target, &TemplateData{
		Frontends:       frontendList,
		PrimaryFrontend: primaryFrontend,
		Backends:        backendList,
		PrimaryBackend:  primaryBackend,
		Env:             getEnvironment(),
	})

	return err
}
