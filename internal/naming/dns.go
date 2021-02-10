/*
 Copyright 2021 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package naming

import (
	"context"
	"net"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
)

// InstancePodDNSNames returns the possible DNS names for instance. The first
// name is the fully qualified domain name (FQDN).
func InstancePodDNSNames(ctx context.Context, instance *appsv1.StatefulSet) []string {
	var (
		domain    = kubernetesClusterDomain(ctx)
		namespace = instance.Namespace
		name      = instance.Name + "-0." + instance.Spec.ServiceName
	)

	// We configure our instances with a subdomain so that Pods get stable DNS
	// names in the form "{pod}.{service}.{namespace}.svc.{cluster-domain}".
	// - https://docs.k8s.io/concepts/services-networking/dns-pod-service/#pods
	return []string{
		name + "." + namespace + ".svc." + domain,
		name + "." + namespace + ".svc",
		name + "." + namespace,
		name,
	}
}

// kubernetesClusterDomain looks up the Kubernetes cluster domain name.
func kubernetesClusterDomain(ctx context.Context) string {
	ctx, span := tracer.Start(ctx, "kubernetes-domain-lookup")
	defer span.End()

	// Lookup an existing Service to determine its fully qualified domain name.
	// This is inexpensive because the "net" package uses OS-level DNS caching.
	// - https://golang.org/issue/24796
	api := "kubernetes.default.svc"
	if cname, err := net.DefaultResolver.LookupCNAME(ctx, api); err == nil {
		return strings.TrimPrefix(cname, api+".")
	} else {
		span.RecordError(err)
	}

	// The kubeadm default is "cluster.local" and is adequate when not running
	// in an actual Kubernetes cluster.
	return "cluster.local."
}
