package kubeapi

import (
	authorizationapi "k8s.io/api/authorization/v1"
	"k8s.io/client-go/kubernetes"
)

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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

// CreateSelfSubjectAccessReview creates a SelfSubjectAccessReview using the ResourceAttributes
// provided
func CreateSelfSubjectAccessReview(clientset *kubernetes.Clientset,
	resourceAttributes *authorizationapi.ResourceAttributes) (*authorizationapi.SelfSubjectAccessReview,
	error) {

	sarRequest := &authorizationapi.SelfSubjectAccessReview{
		Spec: authorizationapi.SelfSubjectAccessReviewSpec{
			ResourceAttributes: resourceAttributes,
		},
	}

	sarResponse, err := clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(sarRequest)
	if err != nil {
		return nil, err
	}

	return sarResponse, nil
}
