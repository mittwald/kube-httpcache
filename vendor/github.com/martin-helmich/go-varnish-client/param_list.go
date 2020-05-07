package varnishclient

import (
	"fmt"
	"regexp"
	"strings"
)

var defaultRegex = regexp.MustCompile(`\s+\(default\)$`)
var unitRegex = regexp.MustCompile(`\s+\[(.*)]$`)

func (c *client) ListParameters() (ParametersResponse, error) {
	resp, err := c.sendRequest("param.show")
	if err != nil {
		return nil, err
	}

	if resp.Code != ResponseOK {
		return nil, fmt.Errorf("could not list parameters (code %d): %s", resp.Code, string(resp.Body))
	}

	lines := strings.Split(string(resp.Body), "\n")
	params := make(ParametersResponse, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		param := Parameter{}

		items := strings.SplitN(line, " ", 2)
		param.Name = items[0]

		if len(items) > 1 {
			val := strings.TrimSpace(items[1])

			if defaultRegex.MatchString(val) {
				param.IsDefault = true
				val = defaultRegex.ReplaceAllString(val, "")
			}

			uMatches := unitRegex.FindStringSubmatch(val)
			if len(uMatches) >= 1 {
				param.Unit = uMatches[1]
				val = unitRegex.ReplaceAllString(val, "")
			}

			param.Value = val
		}

		params = append(params, param)
	}

	return params, nil
}
