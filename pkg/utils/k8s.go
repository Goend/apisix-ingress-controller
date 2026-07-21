// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utils

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/go-logr/logr"

	"github.com/apache/apisix-ingress-controller/internal/types"
)

// HasAPIResource checks if a specific API resource is available in the current cluster.
// It uses the Discovery API to query the cluster's available resources and returns true
// if the resource is found, false otherwise. An error is returned if the discovery API
// is unreachable (e.g. apiserver restart); callers should treat this as a fatal setup
// error rather than silently skipping the resource.
func HasAPIResource(mgr ctrl.Manager, obj client.Object) (bool, error) {
	return HasAPIResourceWithLogger(mgr, obj, ctrl.Log.WithName("api-detection"))
}

// HasAPIResourceWithLogger is the same as HasAPIResource but accepts a custom logger
// for more detailed debugging information.
func HasAPIResourceWithLogger(mgr ctrl.Manager, obj client.Object, logger logr.Logger) (bool, error) {
	gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
	if err != nil {
		logger.Info("cannot derive GVK from scheme", "error", err)
		return false, fmt.Errorf("cannot derive GVK from scheme: %w", err)
	}

	groupVersion := gvk.GroupVersion().String()

	logger = logger.WithValues(
		"kind", gvk.Kind,
		"group", gvk.Group,
		"version", gvk.Version,
		"groupVersion", groupVersion,
	)

	// Create discovery client
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return false, fmt.Errorf("create discovery client: %w", err)
	}

	// Query server resources for the specific group/version.
	// A 404 (IsNotFound) means the entire group/version is not served —
	// the CRD is genuinely absent. Connection errors (refused, timeout, etc.)
	// are returned as an error so callers can fail-fast.
	apiResources, err := discoveryClient.ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("group/version not registered in cluster", "groupVersion", groupVersion)
			return false, nil
		}
		return false, fmt.Errorf("discovery query for %s failed: %w", groupVersion, err)
	}

	// Check if the specific kind exists in the resource list
	for _, res := range apiResources.APIResources {
		if res.Kind == gvk.Kind {
			return true, nil
		}
	}

	logger.Info("API resource kind not found in group/version")
	return false, nil
}

func FormatGVK(obj client.Object) string {
	gvk := types.GvkOf(obj)
	return gvk.String()
}
