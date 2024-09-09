// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package naming

import (
	"context"
	"net"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// InstancePodDNSNames returns the possible DNS names for instance. The first
// name is the fully qualified domain name (FQDN).
func InstancePodDNSNames(ctx context.Context, instance *appsv1.StatefulSet) []string {
	var (
		domain    = KubernetesClusterDomain(ctx)
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

// RepoHostPodDNSNames returns the possible DNS names for a pgBackRest repository host Pod.
// The first name is the fully qualified domain name (FQDN).
func RepoHostPodDNSNames(ctx context.Context, repoHost *appsv1.StatefulSet) []string {
	var (
		domain    = KubernetesClusterDomain(ctx)
		namespace = repoHost.Namespace
		name      = repoHost.Name + "-0." + repoHost.Spec.ServiceName
	)

	// We configure our repository hosts with a subdomain so that Pods get stable
	// DNS names in the form "{pod}.{service}.{namespace}.svc.{cluster-domain}".
	// - https://docs.k8s.io/concepts/services-networking/dns-pod-service/#pods
	return []string{
		name + "." + namespace + ".svc." + domain,
		name + "." + namespace + ".svc",
		name + "." + namespace,
		name,
	}
}

// ServiceDNSNames returns the possible DNS names for service. The first name
// is the fully qualified domain name (FQDN).
func ServiceDNSNames(ctx context.Context, service *corev1.Service) []string {
	domain := KubernetesClusterDomain(ctx)

	return []string{
		service.Name + "." + service.Namespace + ".svc." + domain,
		service.Name + "." + service.Namespace + ".svc",
		service.Name + "." + service.Namespace,
		service.Name,
	}
}

// KubernetesClusterDomain looks up the Kubernetes cluster domain name.
func KubernetesClusterDomain(ctx context.Context) string {
	ctx, span := tracer.Start(ctx, "kubernetes-domain-lookup")
	defer span.End()

	// Lookup an existing Service to determine its fully qualified domain name.
	// This is inexpensive because the "net" package uses OS-level DNS caching.
	// - https://golang.org/issue/24796
	api := "kubernetes.default.svc"
	cname, err := net.DefaultResolver.LookupCNAME(ctx, api)

	if err == nil {
		return strings.TrimPrefix(cname, api+".")
	}

	span.RecordError(err)
	// The kubeadm default is "cluster.local" and is adequate when not running
	// in an actual Kubernetes cluster.
	return "cluster.local."
}
