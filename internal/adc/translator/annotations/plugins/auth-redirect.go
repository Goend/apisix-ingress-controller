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
	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

type authRedirect struct{}

// NewAuthRedirectHandler creates a handler to convert the auth-signin annotation
// to APISIX auth-redirect plugin config. This plugin intercepts 401/403 responses
// (e.g. from the forward-auth plugin) and redirects the client to a signin URL.
func NewAuthRedirectHandler() PluginAnnotationsHandler {
	return &authRedirect{}
}

func (a *authRedirect) PluginName() string {
	return "auth-redirect"
}

func (a *authRedirect) Handle(e annotations.Extractor) (any, error) {
	signinURL := e.GetStringAnnotation(annotations.AnnotationsAuthSigninURL)
	if len(signinURL) == 0 {
		return nil, nil
	}

	return &adctypes.AuthRedirectConfig{
		SigninURL: signinURL,
	}, nil
}
