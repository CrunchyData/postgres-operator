/*
 Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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

package util

import (
	"fmt"

	"k8s.io/component-base/featuregate"
)

const (
	// Every feature gate should add a key here following this template:
	//
	// // Enables FeatureName...
	// FeatureName featuregate.Feature = "FeatureName"
	//
	// - https://releases.k8s.io/v1.20.0/pkg/features/kube_features.go#L27
	//
	// Feature gates should be listed in alphabetical, case-sensitive
	// (upper before any lower case character) order.
	//
	// Enables support of custom sidecars for PostgreSQL instance Pods
	InstanceSidecars featuregate.Feature = "InstanceSidecars"
	//
	// Enables support of custom sidecars for pgBouncer Pods
	PGBouncerSidecars featuregate.Feature = "PGBouncerSidecars"
)

// pgoFeatures consists of all known PGO feature keys.
// To add a new feature, define a key for it above and add it here.
// An example entry is as follows:
//
//	FeatureName: {Default: false, PreRelease: featuregate.Alpha},
//
// - https://releases.k8s.io/v1.20.0/pkg/features/kube_features.go#L729-732
var pgoFeatures = map[featuregate.Feature]featuregate.FeatureSpec{
	InstanceSidecars:  {Default: false, PreRelease: featuregate.Alpha},
	PGBouncerSidecars: {Default: false, PreRelease: featuregate.Alpha},
}

// DefaultMutableFeatureGate is a mutable, shared global FeatureGate.
// It is used to indicate whether a given feature is enabled or not.
//
// - https://pkg.go.dev/k8s.io/apiserver/pkg/util/feature
// - https://releases.k8s.io/v1.20.0/staging/src/k8s.io/apiserver/pkg/util/feature/feature_gate.go#L24-L28
var DefaultMutableFeatureGate featuregate.MutableFeatureGate = featuregate.NewFeatureGate()

// AddAndSetFeatureGates utilizes the Kubernetes feature gate packages to first
// add the default PGO features to the featureGate and then set the values provided
// via the 'PGO_FEATURE_GATES' environment variable. This function expects a string
// like feature1=true,feature2=false,...
//
// - https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
// - https://pkg.go.dev/k8s.io/component-base@v0.20.1/featuregate
func AddAndSetFeatureGates(features string) error {
	// Add PGO features to the featureGate
	// - https://releases.k8s.io/v1.20.0/staging/src/k8s.io/component-base/featuregate/feature_gate.go#L110-L111
	if err := DefaultMutableFeatureGate.Add(pgoFeatures); err != nil {
		return fmt.Errorf("unable to add PGO features to the featureGate. %w", err)
	}

	// Set the feature gates from environment variable config
	// - https://releases.k8s.io/v1.20.0/staging/src/k8s.io/component-base/featuregate/feature_gate.go#L105-L107
	if err := DefaultMutableFeatureGate.Set(features); err != nil {
		return fmt.Errorf("unable to parse and store configured feature gates. %w", err)
	}
	return nil
}
