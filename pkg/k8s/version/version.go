// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

// Package version keeps track of the Kubernetes version the client is
// connected to
package version

import (
	"fmt"
	"log/slog"

	"github.com/blang/semver/v4"
	"k8s.io/client-go/kubernetes"

	"github.com/cilium/cilium/pkg/lock"
	"github.com/cilium/cilium/pkg/versioncheck"
)

// ServerCapabilities is a list of server capabilities derived based on
// version, the Kubernetes discovery API, or probing of individual API
// endpoints.
type ServerCapabilities struct {
	// MinimalVersionMet is true when the minimal version of Kubernetes
	// required to run Cilium has been met
	MinimalVersionMet bool
}

type cachedVersion struct {
	mutex        lock.RWMutex
	capabilities ServerCapabilities
	version      semver.Version
}

const (
	// MinimalVersionConstraint is the minimal version that Cilium supports to
	// run kubernetes.
	MinimalVersionConstraint = "1.21.0"
)

var (
	cached = cachedVersion{}

	// isGEThanMinimalVersionConstraint is the minimal version required to run
	// Cilium
	isGEThanMinimalVersionConstraint = versioncheck.MustCompile(">=" + MinimalVersionConstraint)
)

// Version returns the version of the Kubernetes apiserver
func Version() semver.Version {
	cached.mutex.RLock()
	c := cached.version
	cached.mutex.RUnlock()
	return c
}

// Capabilities returns the capabilities of the Kubernetes apiserver
func Capabilities() ServerCapabilities {
	cached.mutex.RLock()
	c := cached.capabilities
	cached.mutex.RUnlock()
	return c
}

func updateVersion(version semver.Version) {
	cached.mutex.Lock()
	defer cached.mutex.Unlock()

	cached.version = version

	cached.capabilities.MinimalVersionMet = isGEThanMinimalVersionConstraint(version)
}

// Force forces the use of a specific version
func Force(version string) error {
	ver, err := versioncheck.Version(version)
	if err != nil {
		return err
	}
	updateVersion(ver)
	return nil
}

<<<<<<< HEAD
=======
func endpointSlicesFallbackDiscovery(client kubernetes.Interface) error {
	// If a k8s version with discovery v1 is used, then do not even bother
	// checking for v1beta1
	cached.mutex.Lock()
	if cached.capabilities.EndpointSliceV1 {
		cached.capabilities.EndpointSlice = true
		cached.mutex.Unlock()
		return nil
	}
	cached.mutex.Unlock()

	// Discovery of API groups requires the API services of the apiserver to be
	// healthy. Such API services can depend on the readiness of regular pods
	// which require Cilium to function correctly. By treating failure to
	// discover API groups as fatal, a critical loop can be entered in which
	// Cilium cannot start because the API groups can't be discovered.
	//
	// Here we acknowledge the lack of discovery ability as non Fatal and fall back to probing
	// the API directly.
	_, err := client.DiscoveryV1beta1().EndpointSlices("default").Get(context.TODO(), "kubernetes", metav1.GetOptions{})
	if err == nil {
		cached.mutex.Lock()
		cached.capabilities.EndpointSlice = true
		cached.mutex.Unlock()
		return nil
	}

	if errors.IsNotFound(err) {
		log.WithError(err).Info("Unable to retrieve EndpointSlices for default/kubernetes. Disabling EndpointSlices")
		// StatusNotFound is a safe error, EndpointSlices are
		// disabled and the agent can continue.
		return nil
	}

	// Unknown error, we can't derive whether to enable or disable
	// EndpointSlices and need to error out.
	return fmt.Errorf("unable to validate EndpointSlices support: %w", err)
}

func leasesFallbackDiscovery(client kubernetes.Interface, apiDiscoveryEnabled bool) error {
	// apiDiscoveryEnabled is used to fallback leases discovery to directly
	// probing the API when we cannot discover API groups.
	// We require to check for Leases capabilities in operator only, which uses Leases
	// for leader election purposes in HA mode.
	if !apiDiscoveryEnabled {
		log.Debugf("Skipping Leases support fallback discovery")
		return nil
	}

	// Similar to endpointSlicesFallbackDiscovery here we fallback to probing the Kubernetes
	// API directly. `kube-controller-manager` creates a lease in the kube-system namespace
	// and here we try and see if that Lease exists.
	_, err := client.CoordinationV1().Leases("kube-system").Get(context.TODO(), "kube-controller-manager", metav1.GetOptions{})
	if err == nil {
		cached.mutex.Lock()
		cached.capabilities.LeasesResourceLock = true
		cached.mutex.Unlock()
		return nil
	}

	if errors.IsNotFound(err) {
		log.WithError(err).Info("Unable to retrieve Leases for kube-controller-manager. Disabling LeasesResourceLock")
		// StatusNotFound is a safe error, Leases are
		// disabled and the agent can continue
		return nil
	}

	// Unknown error, we can't derive whether to enable or disable
	// LeasesResourceLock and need to error out
	return fmt.Errorf("unable to validate LeasesResourceLock support: %w", err)
}

>>>>>>> 7c898bc70 (spelling: critical)
func updateK8sServerVersion(client kubernetes.Interface) error {
	var ver semver.Version

	sv, err := client.Discovery().ServerVersion()
	if err != nil {
		return err
	}

	// Try GitVersion first. In case of error fallback to MajorMinor
	if sv.GitVersion != "" {
		// This is a string like "v1.9.0"
		ver, err = versioncheck.Version(sv.GitVersion)
		if err == nil {
			updateVersion(ver)
			return nil
		}
	}

	if sv.Major != "" && sv.Minor != "" {
		ver, err = versioncheck.Version(fmt.Sprintf("%s.%s", sv.Major, sv.Minor))
		if err == nil {
			updateVersion(ver)
			return nil
		}
	}

	return fmt.Errorf("cannot parse k8s server version from %+v: %w", sv, err)
}

// Update retrieves the version of the Kubernetes apiserver and derives the
// capabilities. This function must be called after connectivity to the
// apiserver has been established.
//
// Discovery of capabilities only works if the discovery API of the apiserver
// is functional. If it is not available, a warning is logged and the discovery
// falls back to probing individual API endpoints.
func Update(logger *slog.Logger, client kubernetes.Interface, apiDiscoveryEnabled bool) error {
	err := updateK8sServerVersion(client)
	if err != nil {
		return err
	}

	return nil
}
