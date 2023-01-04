<!--
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->


## Feature Gates

Feature gates allow users to enable or disable
certain features by setting the "PGO_FEATURE_GATES" environment
variable to a list similar to "feature1=true,feature2=false,..."
in the PGO Deployment.

This capability leverages the relevant Kubernetes packages. Documentation and
code implementation examples are given below.

- Documentation:
  - https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/

- Package Information:
  - https://pkg.go.dev/k8s.io/component-base@v0.20.1/featuregate

- Adding the feature gate key:
  - https://releases.k8s.io/v1.20.0/pkg/features/kube_features.go#L27

- Adding the feature gate to the known features map:
  - https://releases.k8s.io/v1.20.0/pkg/features/kube_features.go#L729-732

- Adding features to the featureGate
  - https://releases.k8s.io/v1.20.0/staging/src/k8s.io/component-base/featuregate/feature_gate.go#L110-L111

- Setting the feature gates
  - https://releases.k8s.io/v1.20.0/staging/src/k8s.io/component-base/featuregate/feature_gate.go#L105-L107

## Developing with Feature Gates in PGO

To add a new feature gate, a few steps are required. First, in
`internal/util/features.go`, you will add a feature gate key name. As an example,
for a new feature called 'FeatureName', you would add a new constant and comment
describing what the feature gate controls at the top of the file, similar to
```
// Enables FeatureName in PGO
FeatureName featuregate.Feature = "FeatureName"
```

Next, add a new entry to the `pgoFeatures` map
```
var pgoFeatures = map[featuregate.Feature]featuregate.FeatureSpec{
    FeatureName: {Default: false, PreRelease: featuregate.Alpha},
}
```
where `FeatureName` is the constant defined previously, `Default: false` sets the
default behavior and `PreRelease: featuregate.Alpha`. The possible `PreRelease`
values are `Alpha`, `Beta`, `GA` and `Deprecated`.

- https://pkg.go.dev/k8s.io/component-base@v0.20.1/featuregate#pkg-constants

By Kubernetes convention, `Alpha` features have almost always been disabled by
default. `Beta` features are generally enabled by default.

- https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-stages

Prior to Kubernetes 1.24, both `Beta` features and APIs were enabled by default.
Starting in v1.24, new `Beta` APIs are generally disabled by default, while `Beta`
features remain enabled by default.

- https://kubernetes.io/blog/2021/07/14/upcoming-changes-in-kubernetes-1-22/#kubernetes-api-removals
- https://kubernetes.io/blog/2022/05/03/kubernetes-1-24-release-announcement/#beta-apis-off-by-default
- https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/3136-beta-apis-off-by-default#goals

For consistency with Kubernetes, we recommend that feature-gated features be
configured as `Alpha` and disabled by default. Any `Beta` features added should
stay consistent with Kubernetes practice and be enabled by default, but we should
keep an eye out for changes to these standards and adjust as needed.

Once the above items are set, you can then use your feature gated value in the
code base to control feature behavior using something like
```
if util.DefaultMutableFeatureGate.Enabled(util.FeatureName)
```

To test the feature gate, set the `PGO_FEATURE_GATES` environment variable to
enable the new feature as follows
```
PGO_FEATURE_GATES="FeatureName=true"
```
Note that for more than one feature, this variable accepts a comma delimited
list, e.g.
```
PGO_FEATURE_GATES="FeatureName=true,FeatureName2=true,FeatureName3=true"
```

While `PGO_FEATURE_GATES` does not have to be set, please note that the features
must be defined before use, otherwise PGO deployment will fail with the
following message
`panic: unable to parse and store configured feature gates. unrecognized feature gate`

Also, the features must have boolean values, otherwise you will see
`panic: unable to parse and store configured feature gates. invalid value`

When dealing with tests that do not invoke `cmd/postgres-operator/main.go`, keep
in mind that you will need to ensure that you invoke the `AddAndSetFeatureGates`
function. Otherwise, any test that references the undefined feature gate will fail
with a panic message similar to
"feature "FeatureName" is not registered in FeatureGate"

To correct for this, you simply need a line similar to
```
err := util.AddAndSetFeatureGates("")
```
