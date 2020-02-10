package apiserver

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
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/ns"
	"github.com/crunchydata/postgres-operator/tlsutil"
	"github.com/crunchydata/postgres-operator/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const rsaKeySize = 2048
const duration365d = time.Hour * 24 * 365
const PGOSecretName = "pgo.tls"

const VERSION_MISMATCH_ERROR = "pgo client and server version mismatch"

// RESTClient ...
var RESTClient *rest.RESTClient

// Clientset ...
var Clientset *kubernetes.Clientset
var RESTConfig *rest.Config

// MetricsFlag if set to true will cause crunchy-collect to be added into new clusters
var MetricsFlag, BadgerFlag bool

// AuditFlag if set to true will cause auditing to occur in the logs
var AuditFlag bool

// DebugFlag is the debug flag value
var DebugFlag bool

// BasicAuth comes from the apiserver config
var BasicAuth bool

// Namespace comes from the apiserver config in this version
var PgoNamespace string
var InstallationName string

var CRUNCHY_DEBUG bool

// TreeTrunk is for debugging only in this context
const TreeTrunk = "└── "

// TreeBranch is for debugging only in this context
const TreeBranch = "├── "

type CredentialDetail struct {
	Username   string
	Password   string
	Role       string
	Namespaces []string
}

var Pgo config.PgoConfig

type containerResourcesTemplateFields struct {
	RequestsMemory, RequestsCPU string
	LimitsMemory, LimitsCPU     string
}

func Initialize() {

	PgoNamespace = os.Getenv("PGO_OPERATOR_NAMESPACE")
	if PgoNamespace == "" {
		log.Info("PGO_OPERATOR_NAMESPACE environment variable is not set and is required, this is the namespace that the Operator is to run within.")
		os.Exit(2)
	}
	log.Info("Pgo Namespace is [" + PgoNamespace + "]")

	//namespaceList := util.GetNamespaces()
	//log.Debugf("watching the following namespaces: [%v]", namespaceList)

	InstallationName = os.Getenv("PGO_INSTALLATION_NAME")
	if InstallationName == "" {
		log.Error("PGO_INSTALLATION_NAME environment variable is missng")
		os.Exit(2)
	}
	log.Info("InstallationName is [" + InstallationName + "]")

	tmp := os.Getenv("CRUNCHY_DEBUG")
	CRUNCHY_DEBUG = false
	if tmp == "true" {
		CRUNCHY_DEBUG = true
	}
	BasicAuth = true
	MetricsFlag = false
	BadgerFlag = false
	AuditFlag = false

	log.Infoln("apiserver starts")

	ConnectToKube()

	InitializePerms()

	err := Pgo.GetConfig(Clientset, PgoNamespace)
	if err != nil {
		log.Error(err)
		log.Error("error in Pgo configuration")
		os.Exit(2)
	}

	initConfig()

	validateWithKube()

	//validateUserCredentials()
}

// ConnectToKube ...
func ConnectToKube() {

	var err error
	RESTConfig, Clientset, err = kubeapi.NewControllerClient()
	if err != nil {
		panic(err)
	}

	RESTClient, _, err = util.NewClient(RESTConfig)
	if err != nil {
		panic(err)
	}

}

func initConfig() {

	AuditFlag = Pgo.Pgo.Audit
	if AuditFlag {
		log.Info("audit flag is set to true")
	}

	MetricsFlag = Pgo.Cluster.Metrics
	if MetricsFlag {
		log.Info("metrics flag is set to true")
	}
	BadgerFlag = Pgo.Cluster.Badger
	if BadgerFlag {
		log.Info("badger flag is set to true")
	}

	tmp := Pgo.BasicAuth
	if tmp == "" {
		BasicAuth = true
	} else {
		var err error
		BasicAuth, err = strconv.ParseBool(tmp)
		if err != nil {
			log.Error("BasicAuth config value is not valid")
			os.Exit(2)
		}
	}
	log.Infof("BasicAuth is %v", BasicAuth)

	if !validStorageSettings() {
		log.Error("Storage Settings are not defined correctly, can't continue")
		os.Exit(2)
	}

	if !validContainerResourcesSettings() {
		log.Error("Container Resources settings are not defined correctly, can't continue")
		os.Exit(2)
	}

}

func BasicAuthCheck(username, password string) bool {

	if BasicAuth == false {
		return true
	}

	//see if there is a pgouser Secret for this username
	secretName := "pgouser-" + username
	secret, found, _ := kubeapi.GetSecret(Clientset, secretName, PgoNamespace)
	if !found {
		log.Errorf("%s username Secret is not found", username)
		return false
	}

	psw := string(secret.Data["password"])
	if psw != password {
		log.Errorf("%s  %s password does not match for user %s ", psw, password, username)
		return false
	}

	return true
}

func BasicAuthzCheck(username, perm string) bool {

	secretName := "pgouser-" + username
	secret, found, _ := kubeapi.GetSecret(Clientset, secretName, PgoNamespace)
	if !found {
		log.Errorf("%s username Secret is not found", username)
		return false
	}

	//get the roles for this user
	rolesString := string(secret.Data["roles"])
	roles := strings.Split(rolesString, ",")
	if len(roles) == 0 {
		log.Errorf("%s user has no roles ", username)
		return false
	}

	//venture thru each role this user has looking for a perm match
	for _, r := range roles {

		//get the pgorole
		roleSecretName := "pgorole-" + r
		rolesecret, found, _ := kubeapi.GetSecret(Clientset, roleSecretName, PgoNamespace)
		if !found {
			log.Errorf("%s pgorole Secret is not found for user %s", r, username)
			return false
		}

		permsString := strings.TrimSpace(string(rolesecret.Data["permissions"]))

		// first a special case. If this is a solitary "*" indicating that this
		// encompasses every permission, then we can exit here as true
		if permsString == "*" {
			return true
		}

		// otherwise, blow up the permission string and see if the user has explicit
		// permission (i.e. is authorized) to access this resource
		perms := strings.Split(permsString, ",")

		for _, p := range perms {
			pp := strings.TrimSpace(p)
			if pp == perm {
				log.Debugf("%s perm found in role %s for username %s", pp, r, username)
				return true
			}
		}

	}

	return false

}

//GetNamespace determines if a user has permission for
//a namespace they are requesting
//a valid requested namespace is required
func GetNamespace(clientset *kubernetes.Clientset, username, requestedNS string) (string, error) {

	log.Debugf("GetNamespace username [%s] ns [%s]", username, requestedNS)

	if requestedNS == "" {
		return requestedNS, errors.New("empty namespace is not valid from pgo clients")
	}

	iAccess, uAccess := UserIsPermittedInNamespace(username, requestedNS)
	if uAccess == false {
		errMsg := fmt.Sprintf("user [%s] is not allowed access to namespace [%s]", username, requestedNS)
		return requestedNS, errors.New(errMsg)
	}
	if iAccess == false {
		errMsg := fmt.Sprintf("namespace [%s] is not part of the Operator installation", requestedNS)
		return requestedNS, errors.New(errMsg)
	}

	if ns.WatchingNamespace(clientset, requestedNS, InstallationName) {
		return requestedNS, nil
	}

	log.Debugf("GetNamespace did not find the requested namespace %s", requestedNS)
	return requestedNS, errors.New("requested Namespace was not found to be in the list of Namespaces being watched.")
}

// Authn performs HTTP Basic Authentication against a user if "BasicAuth" is set
// to "true" (which it is by default).
//
// ...it also performs Authorization (Authz) against the user that is attempting
// to authenticate, and as such, to truly "authenticate/authorize," one needs
// at least a valid Operator User account.
func Authn(perm string, w http.ResponseWriter, r *http.Request) (string, error) {
	var err error
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

	// Need to run the HTTP library `BasicAuth` even if `BasicAuth == false`, as
	// this function currently encapsulates authorization as well, and this is
	// the call where we get the username to check the RBAC settings
	username, password, authOK := r.BasicAuth()
	if AuditFlag {
		log.Infof("[audit] %s username=[%s] method=[%s] ip=[%s] ok=[%t] ", perm, username, r.Method, r.RemoteAddr, authOK)
	}

	// Check to see if this user is authenticated
	// If BasicAuth is "disabled", skip the authentication; o/w/ check if the
	// authentication passed
	if !BasicAuth {
		log.Debugf("BasicAuth disabled, Skipping Authentication %s username=[%s]", perm, username)
	} else {
		log.Debugf("Authentication Attempt %s username=[%s]", perm, username)
		if !authOK {
			http.Error(w, "Not Authorized. Basic Authentication credentials must be provided according to RFC 7617, Section 2.", 401)
			return "", errors.New("Not Authorized: Credentials do not comply with RFC 7617")
		}
	}

	if !BasicAuthCheck(username, password) {
		log.Errorf("Authentication Failed %s username=[%s]", perm, username)
		http.Error(w, "Not authenticated in apiserver", 401)
		return "", errors.New("Not Authenticated")
	}

	if !BasicAuthzCheck(username, perm) {
		log.Errorf("Authorization Failed %s username=[%s]", perm, username)
		http.Error(w, "Not authorized for this apiserver action", 403)
		return "", errors.New("Not authorized for this apiserver action")
	}

	log.Debug("Authentication Success")
	return username, err

}

func validContainerResourcesSettings() bool {
	log.Infof("ContainerResources has %d definitions", len(Pgo.ContainerResources))

	//validate any Container Resources in pgo.yaml for correct formats
	if !IsValidContainerResourceValues() {
		return false
	}

	drs := Pgo.DefaultContainerResources
	if drs == "" {
		log.Info("DefaultContainerResources was not specified in pgo.yaml, so no container resources will be specified")
		return true
	}

	//validate the DefaultContainerResource value
	if IsValidContainerResource(drs) {
		log.Info(drs + " is valid")
	} else {
		log.Error(drs + " is NOT valid")
		return false
	}

	return true

}

func validStorageSettings() bool {
	log.Infof("Storage has %d definitions", len(Pgo.Storage))

	ps := Pgo.PrimaryStorage
	if IsValidStorageName(ps) {
		log.Info(ps + " is valid")
	} else {
		log.Error(ps + " is NOT valid")
		return false
	}
	rs := Pgo.ReplicaStorage
	if IsValidStorageName(rs) {
		log.Info(rs + " is valid")
	} else {
		log.Error(rs + " is NOT valid")
		return false
	}
	bs := Pgo.BackupStorage
	if IsValidStorageName(bs) {
		log.Info(bs + " is valid")
	} else {
		log.Error(bs + " is NOT valid")
		return false
	}

	return true

}

func IsValidContainerResource(name string) bool {
	_, ok := Pgo.ContainerResources[name]
	return ok
}

func IsValidStorageName(name string) bool {
	_, ok := Pgo.Storage[name]
	return ok
}

// ValidateNodeLabel
// returns error if node label is invalid
func ValidateNodeLabel(nodeLabel string) error {
	parts := strings.Split(nodeLabel, "=")
	if len(parts) != 2 {
		return errors.New(nodeLabel + " node label does not follow key=value format")
	}

	keyValid, valueValid, err := IsValidNodeLabel(parts[0], parts[1])
	if err != nil {
		return err
	}

	if !keyValid {
		return errors.New(nodeLabel + " key was not valid .. check node labels for correct values to specify")
	}
	if !valueValid {
		return errors.New(nodeLabel + " node label value was not valid .. check node labels for correct values to specify")
	}

	return nil
}

// IsValidNodeLabel
// returns bool for key validity
// returns bool for value validity
// returns error
func IsValidNodeLabel(key, value string) (bool, bool, error) {

	var err error
	keyValid := false
	valueValid := false

	nodes, err := kubeapi.GetAllNodes(Clientset)
	if err != nil {
		return false, false, err
	}

	for _, node := range nodes.Items {

		if val, exists := node.ObjectMeta.Labels[key]; exists {
			keyValid = true
			if val == value {
				valueValid = true
			}
		}
	}

	return keyValid, valueValid, err
}

func IsValidContainerResourceValues() bool {

	var err error

	for k, v := range Pgo.ContainerResources {
		log.Infof("Container Resources %s [%v]", k, v)
		resources, _ := Pgo.GetContainerResource(k)
		_, err = resource.ParseQuantity(resources.RequestsMemory)
		if err != nil {
			log.Errorf("%s.RequestsMemory value invalid format", k)
			return false
		}
		_, err = resource.ParseQuantity(resources.RequestsCPU)
		if err != nil {
			log.Errorf("%s.RequestsCPU value invalid format", k)
			return false
		}
		_, err = resource.ParseQuantity(resources.LimitsMemory)
		if err != nil {
			log.Errorf("%s.LimitsMemory value invalid format", k)
			return false
		}
		_, err = resource.ParseQuantity(resources.LimitsCPU)
		if err != nil {
			log.Errorf("%s.LimitsCPU value invalid format", k)
			return false
		}
	}
	return true
}

func validateWithKube() {
	log.Debug("validateWithKube called")

	configNodeLabels := make([]string, 2)
	configNodeLabels[0] = Pgo.Cluster.PrimaryNodeLabel
	configNodeLabels[1] = Pgo.Cluster.ReplicaNodeLabel

	for _, n := range configNodeLabels {

		//parse & validate pgo.yaml node labels if set
		if n != "" {

			if err := ValidateNodeLabel(n); err != nil {
				log.Error(n + " node label specified in pgo.yaml is invalid")
				log.Error(err)
				os.Exit(2)
			}

			log.Debugf("%s is a valid pgo.yaml node label default", n)
		}
	}

	err := ns.ValidateNamespaces(Clientset, InstallationName, PgoNamespace)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}

}

// GetContainerResources ...
func GetContainerResourcesJSON(resources *crv1.PgContainerResources) string {

	//test for the case where no container resources are specified
	if resources.RequestsMemory == "" || resources.RequestsCPU == "" ||
		resources.LimitsMemory == "" || resources.LimitsCPU == "" {
		return ""
	}
	fields := containerResourcesTemplateFields{}
	fields.RequestsMemory = resources.RequestsMemory
	fields.RequestsCPU = resources.RequestsCPU
	fields.LimitsMemory = resources.LimitsMemory
	fields.LimitsCPU = resources.LimitsCPU

	doc := bytes.Buffer{}
	err := config.ContainerResourcesTemplate.Execute(&doc, fields)
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	if log.GetLevel() == log.DebugLevel {
		config.ContainerResourcesTemplate.Execute(os.Stdout, fields)
	}

	return doc.String()
}

//returns installation access and user access
//installation access means a namespace belongs to this Operator installation
//user access means this user has access to a namespace
func UserIsPermittedInNamespace(username, requestedNS string) (bool, bool) {

	iAccess := false
	uAccess := false

	ns, found, err := kubeapi.GetNamespace(Clientset, requestedNS)
	if !found {
		log.Error(err)
		log.Errorf("could not find namespace %s ", requestedNS)
		return iAccess, uAccess
	}

	if ns.ObjectMeta.Labels[config.LABEL_VENDOR] == config.LABEL_CRUNCHY &&

		ns.ObjectMeta.Labels[config.LABEL_PGO_INSTALLATION_NAME] == InstallationName {
		iAccess = true

	}

	//get the pgouser Secret for this username
	userSecretName := "pgouser-" + username
	userSecret, found, err := kubeapi.GetSecret(Clientset, userSecretName, PgoNamespace)
	if !found {
		uAccess = false
		log.Error(err)
		log.Errorf("could not find pgouser Secret for username %s", username)
		return iAccess, uAccess
	}

	nsstring := string(userSecret.Data["namespaces"])
	nsList := strings.Split(nsstring, ",")
	for _, v := range nsList {
		ns := strings.TrimSpace(v)
		if ns == requestedNS {
			uAccess = true
			return iAccess, uAccess
		}
	}

	//handle the case of a user in pgouser with "" (all) namespaces
	if nsstring == "" {
		uAccess = true
		return iAccess, uAccess
	}

	uAccess = false
	return iAccess, uAccess
}

// WriteTLSCert writes the server certificate and key to files from the
// PGOSecretName secret or generates a new key (writing to both the secret
// and the expected files
func WriteTLSCert(certPath, keyPath string) error {

	var pgoSecret *v1.Secret
	var found bool
	var err error

	pgoSecret, found, err = kubeapi.GetSecret(Clientset, PGOSecretName, PgoNamespace)
	if found {
		log.Infof("%s Secret found in namespace %s", PGOSecretName, PgoNamespace)
		log.Infof("cert key data len is %d", len(pgoSecret.Data[v1.TLSCertKey]))
		if err := ioutil.WriteFile(certPath, pgoSecret.Data[v1.TLSCertKey], 0644); err != nil {
			return err
		}
		log.Infof("private key data len is %d", len(pgoSecret.Data[v1.TLSPrivateKeyKey]))
		if err := ioutil.WriteFile(keyPath, pgoSecret.Data[v1.TLSPrivateKeyKey], 0644); err != nil {
			return err
		}
	} else {
		log.Infof("%s Secret NOT found in namespace %s", PGOSecretName, PgoNamespace)
		err = generateTLSCert(certPath, keyPath)
		if err != nil {
			log.Error("error generating pgo.tls Secret")
			return err
		}
	}

	return nil

}

// generateTLSCert generates a self signed cert and stores it in both
// the PGOSecretName Secret and certPath, keyPath files
func generateTLSCert(certPath, keyPath string) error {
	var err error

	//generate private key
	var privateKey *rsa.PrivateKey
	privateKey, err = tlsutil.NewPrivateKey()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	privateKeyBytes := tlsutil.EncodePrivateKeyPEM(privateKey)
	log.Debugf("generated privateKeyBytes len %d", len(privateKeyBytes))

	var caCert *x509.Certificate
	caCert, err = tlsutil.NewSelfSignedCACertificate(privateKey)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	caCertBytes := tlsutil.EncodeCertificatePEM(caCert)
	log.Debugf("generated caCertBytes len %d", len(caCertBytes))

	// CreateSecret
	newSecret := v1.Secret{}
	newSecret.Name = PGOSecretName
	newSecret.ObjectMeta.Labels = make(map[string]string)
	newSecret.ObjectMeta.Labels[config.LABEL_VENDOR] = "crunchydata"
	newSecret.Data = make(map[string][]byte)
	newSecret.Data[v1.TLSCertKey] = caCertBytes
	newSecret.Data[v1.TLSPrivateKeyKey] = privateKeyBytes
	newSecret.Type = v1.SecretTypeTLS

	err = kubeapi.CreateSecret(Clientset, &newSecret, PgoNamespace)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	if err := ioutil.WriteFile(certPath, newSecret.Data[v1.TLSCertKey], 0644); err != nil {
		return err
	}
	if err := ioutil.WriteFile(keyPath, newSecret.Data[v1.TLSPrivateKeyKey], 0644); err != nil {
		return err
	}

	return err

}
