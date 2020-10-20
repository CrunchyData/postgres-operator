package backrest

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/operator"
	"github.com/crunchydata/postgres-operator/internal/operator/pvc"
	"github.com/crunchydata/postgres-operator/internal/util"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// s3RepoTypeRegex defines a regex to detect if an S3 restore has been specified using the
// pgBackRest --repo-type option
var s3RepoTypeRegex = regexp.MustCompile(`--repo-type=["']?s3["']?`)

type RepoDeploymentTemplateFields struct {
	SecurityContext           string
	PGOImagePrefix            string
	PGOImageTag               string
	ContainerResources        string
	BackrestRepoClaimName     string
	SshdSecretsName           string
	PGbackrestDBHost          string
	PgbackrestRepoPath        string
	PgbackrestDBPath          string
	PgbackrestPGPort          string
	SshdPort                  int
	PgbackrestStanza          string
	PgbackrestRepoType        string
	PgbackrestS3EnvVars       string
	Name                      string
	ClusterName               string
	PodAnnotations            string
	PodAntiAffinity           string
	PodAntiAffinityLabelName  string
	PodAntiAffinityLabelValue string
	Replicas                  int
	BootstrapCluster          string
}

type RepoServiceTemplateFields struct {
	Name        string
	ClusterName string
	Port        string
}

// CreateRepoDeployment creates a pgBackRest repository deployment for a PostgreSQL cluster,
// while also creating the associated Service and PersistentVolumeClaim.
func CreateRepoDeployment(clientset kubernetes.Interface, cluster *crv1.Pgcluster,
	createPVC, bootstrapRepo bool, replicas int) error {
	ctx := context.TODO()

	namespace := cluster.GetNamespace()
	restoreClusterName := cluster.Spec.PGDataSource.RestoreFrom

	repoFields := getRepoDeploymentFields(clientset, cluster, replicas)

	var repoName, serviceName string
	// if this is a bootstrap repository then we now override certain fields as needed
	if bootstrapRepo {
		if err := setBootstrapRepoOverrides(clientset, cluster, repoFields); err != nil {
			return err
		}
		repoName = fmt.Sprintf(util.BackrestRepoPVCName, restoreClusterName)
		serviceName = fmt.Sprintf(util.BackrestRepoServiceName, restoreClusterName)
	} else {
		repoName = fmt.Sprintf(util.BackrestRepoPVCName, cluster.Name)
		serviceName = fmt.Sprintf(util.BackrestRepoServiceName, cluster.Name)
	}

	//create backrest repo service
	serviceFields := RepoServiceTemplateFields{
		Name:        serviceName,
		ClusterName: cluster.Name,
		Port:        "2022",
	}

	err := createService(clientset, &serviceFields, namespace)
	if err != nil {
		log.Error(err)
		return err
	}

	// if createPVC is set to true, attempt to create the PVC
	if createPVC {
		// create backrest repo PVC with same name as repoName
		_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, repoName, metav1.GetOptions{})
		if err == nil {
			log.Debugf("pvc [%s] already present, will not recreate", repoName)
		} else if kerrors.IsNotFound(err) {
			_, err = pvc.CreatePVC(clientset, &cluster.Spec.BackrestStorage, repoName, cluster.Name, namespace)
			if err != nil {
				return err
			}
			log.Debugf("created backrest-shared-repo pvc [%s]", repoName)
		} else {
			return err
		}
	}

	var b bytes.Buffer
	err = config.PgoBackrestRepoTemplate.Execute(&b, repoFields)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if operator.CRUNCHY_DEBUG {
		config.PgoBackrestRepoTemplate.Execute(os.Stdout, repoFields)
	}

	deployment := appsv1.Deployment{}
	err = json.Unmarshal(b.Bytes(), &deployment)
	if err != nil {
		log.Error("error unmarshalling backrest repo json into Deployment " + err.Error())
		return err
	}

	operator.AddBackRestConfigVolumeAndMounts(&deployment.Spec.Template.Spec, cluster.Name, cluster.Spec.BackrestConfig)

	// set the container image to an override value, if one exists
	operator.SetContainerImageOverride(config.CONTAINER_IMAGE_PGO_BACKREST_REPO,
		&deployment.Spec.Template.Spec.Containers[0])

	if _, err := clientset.AppsV1().Deployments(namespace).Create(ctx, &deployment, metav1.CreateOptions{}); err != nil &&
		!kerrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// setBootstrapRepoOverrides overrides certain fields used to populate the pgBackRest repository template
// as needed to support the creation of a bootstrap repository need to bootstrap a new cluster from an
// existing data source.
func setBootstrapRepoOverrides(clientset kubernetes.Interface, cluster *crv1.Pgcluster,
	repoFields *RepoDeploymentTemplateFields) error {
	ctx := context.TODO()

	restoreClusterName := cluster.Spec.PGDataSource.RestoreFrom
	namespace := cluster.GetNamespace()

	repoFields.ClusterName = restoreClusterName
	repoFields.BootstrapCluster = cluster.GetName()
	repoFields.Name = fmt.Sprintf(util.BackrestRepoServiceName, restoreClusterName)
	repoFields.SshdSecretsName = fmt.Sprintf(util.BackrestRepoSecretName, restoreClusterName)

	// set the proper PVC name for the "restore from" cluster
	repoFields.BackrestRepoClaimName = fmt.Sprintf(util.BackrestRepoPVCName, restoreClusterName)

	restoreFromSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx,
		fmt.Sprintf("%s-%s", restoreClusterName, config.LABEL_BACKREST_REPO_SECRET),
		metav1.GetOptions{})
	if err != nil {
		return err
	}

	repoFields.PgbackrestRepoPath = restoreFromSecret.Annotations[config.ANNOTATION_REPO_PATH]
	repoFields.PgbackrestPGPort = restoreFromSecret.Annotations[config.ANNOTATION_PG_PORT]

	sshdPort, err := strconv.Atoi(restoreFromSecret.Annotations[config.ANNOTATION_SSHD_PORT])
	if err != nil {
		return err
	}
	repoFields.SshdPort = sshdPort

	// if an s3 restore is detected, override or set the pgbackrest S3 env vars, otherwise do
	// not set the s3 env vars at all
	s3Restore := S3RepoTypeCLIOptionExists(cluster.Spec.PGDataSource.RestoreOpts)
	if s3Restore {
		// Now override any backrest S3 env vars for the bootstrap job
		repoFields.PgbackrestS3EnvVars = operator.GetPgbackrestBootstrapS3EnvVars(
			cluster.Spec.PGDataSource.RestoreFrom, restoreFromSecret)
	} else {
		repoFields.PgbackrestS3EnvVars = ""
	}

	return nil
}

// getRepoDeploymentFields returns a RepoDeploymentTemplateFields struct populated with the fields
// needed to populate the pgBackRest repository template and create a pgBackRest repository for a
// specific PostgreSQL cluster.
func getRepoDeploymentFields(clientset kubernetes.Interface, cluster *crv1.Pgcluster,
	replicas int) *RepoDeploymentTemplateFields {

	namespace := cluster.GetNamespace()

	repoFields := RepoDeploymentTemplateFields{
		PGOImagePrefix:        util.GetValueOrDefault(cluster.Spec.PGOImagePrefix, operator.Pgo.Pgo.PGOImagePrefix),
		PGOImageTag:           operator.Pgo.Pgo.PGOImageTag,
		ContainerResources:    operator.GetResourcesJSON(cluster.Spec.BackrestResources, cluster.Spec.BackrestLimits),
		BackrestRepoClaimName: fmt.Sprintf(util.BackrestRepoPVCName, cluster.Name),
		SshdSecretsName:       fmt.Sprintf(util.BackrestRepoSecretName, cluster.Name),
		PGbackrestDBHost:      cluster.Name,
		PgbackrestRepoPath:    util.GetPGBackRestRepoPath(*cluster),
		PgbackrestDBPath:      "/pgdata/" + cluster.Name,
		PgbackrestPGPort:      cluster.Spec.Port,
		SshdPort:              operator.Pgo.Cluster.BackrestPort,
		PgbackrestStanza:      "db",
		PgbackrestRepoType:    operator.GetRepoType(cluster.Spec.UserLabels[config.LABEL_BACKREST_STORAGE_TYPE]),
		PgbackrestS3EnvVars:   operator.GetPgbackrestS3EnvVars(*cluster, clientset, namespace),
		Name:                  fmt.Sprintf(util.BackrestRepoServiceName, cluster.Name),
		ClusterName:           cluster.Name,
		SecurityContext:       operator.GetPodSecurityContext(cluster.Spec.BackrestStorage.GetSupplementalGroups()),
		Replicas:              replicas,
		PodAnnotations:        operator.GetAnnotations(cluster, crv1.ClusterAnnotationBackrest),
		PodAntiAffinity: operator.GetPodAntiAffinity(cluster,
			crv1.PodAntiAffinityDeploymentPgBackRest, cluster.Spec.PodAntiAffinity.PgBackRest),
		PodAntiAffinityLabelName: config.LABEL_POD_ANTI_AFFINITY,
		PodAntiAffinityLabelValue: string(operator.GetPodAntiAffinityType(cluster,
			crv1.PodAntiAffinityDeploymentPgBackRest, cluster.Spec.PodAntiAffinity.PgBackRest)),
	}

	return &repoFields
}

// UpdateAnnotations updates the annotations in the "template" portion of a
// pgBackRest deployment
func UpdateAnnotations(clientset kubernetes.Interface, cluster *crv1.Pgcluster,
	annotations map[string]string) error {
	ctx := context.TODO()

	// get a list of all of the instance deployments for the cluster
	deployment, err := operator.GetBackrestDeployment(clientset, cluster)

	if err != nil {
		return err
	}

	// now update the pgBackRest deployment
	log.Debugf("update annotations on [%s]", deployment.Name)
	log.Debugf("new annotations: %v", annotations)

	deployment.Spec.Template.SetAnnotations(annotations)

	// finally, update the Deployment. If something errors, we'll log that there
	// was an error, but continue with processing the other deployments
	_, err = clientset.AppsV1().Deployments(deployment.Namespace).
		Update(ctx, deployment, metav1.UpdateOptions{})

	return err
}

// UpdateResources updates the pgBackRest repository Deployment to reflect any
// resource updates
func UpdateResources(clientset kubernetes.Interface, cluster *crv1.Pgcluster) error {
	ctx := context.TODO()

	// get a list of all of the instance deployments for the cluster
	deployment, err := operator.GetBackrestDeployment(clientset, cluster)

	if err != nil {
		return err
	}

	// first, initialize the requests/limits resource to empty Resource Lists
	deployment.Spec.Template.Spec.Containers[0].Resources.Requests = v1.ResourceList{}
	deployment.Spec.Template.Spec.Containers[0].Resources.Limits = v1.ResourceList{}

	// now, simply deep copy the values from the CRD
	if cluster.Spec.BackrestResources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources.Requests = cluster.Spec.BackrestResources.DeepCopy()
	}

	if cluster.Spec.BackrestLimits != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources.Limits = cluster.Spec.BackrestLimits.DeepCopy()
	}

	// update the deployment with the new values
	_, err = clientset.AppsV1().Deployments(deployment.Namespace).
		Update(ctx, deployment, metav1.UpdateOptions{})

	return err
}

func createService(clientset kubernetes.Interface, fields *RepoServiceTemplateFields, namespace string) error {
	ctx := context.TODO()

	var err error

	var b bytes.Buffer

	_, err = clientset.CoreV1().Services(namespace).Get(ctx, fields.Name, metav1.GetOptions{})
	if err != nil {

		err = config.PgoBackrestRepoServiceTemplate.Execute(&b, fields)
		if err != nil {
			log.Error(err.Error())
			return err
		}

		if operator.CRUNCHY_DEBUG {
			config.PgoBackrestRepoServiceTemplate.Execute(os.Stdout, fields)
		}

		s := v1.Service{}
		err = json.Unmarshal(b.Bytes(), &s)
		if err != nil {
			log.Error("error unmarshalling repo service json into repo Service " + err.Error())
			return err
		}

		_, err = clientset.CoreV1().Services(namespace).Create(ctx, &s, metav1.CreateOptions{})
	}

	return err
}

// S3RepoTypeCLIOptionExists detects if a S3 restore was requested via the '--repo-type'
// command line option
func S3RepoTypeCLIOptionExists(opts string) bool {
	return s3RepoTypeRegex.MatchString(opts)
}
