// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package plugins

import (
	"strconv"
	"strings"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

type customErrorPage struct{}

// NewCustomErrorPageHandler creates a handler to convert the custom-error-codes annotation
// to APISIX custom-error-page plugin config. This plugin intercepts specified HTTP error
// status codes and returns custom error pages instead of upstream error responses.
// The actual page content is configured via plugin metadata (global config).
func NewCustomErrorPageHandler() PluginAnnotationsHandler {
	return &customErrorPage{}
}

func (c *customErrorPage) PluginName() string {
	return "custom-error-page"
}

func (c *customErrorPage) Handle(e annotations.Extractor) (any, error) {
	codesStr := e.GetStringAnnotation(annotations.AnnotationsCustomErrorCodes)
	if len(codesStr) == 0 {
		return nil, nil
	}

	codes := parseStatusCodes(codesStr)
	if len(codes) == 0 {
		return nil, nil
	}

	return &adctypes.CustomErrorPageConfig{
		Enable: true,
		Codes:  codes,
	}, nil
}

// parseStatusCodes parses a comma-separated string of HTTP status codes into a slice of ints.
// Invalid codes are silently skipped.
func parseStatusCodes(s string) []int {
	parts := strings.Split(s, ",")
	var codes []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		code, err := strconv.Atoi(p)
		if err != nil || code < 400 || code > 599 {
			continue
		}
		codes = append(codes, code)
	}
	return codes
}
