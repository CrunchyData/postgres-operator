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
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/kubeapi"
	"github.com/crunchydata/postgres-operator/internal/ns"
	"github.com/crunchydata/postgres-operator/internal/tlsutil"
	"github.com/crunchydata/postgres-operator/internal/util"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const PGOSecretName = "pgo.tls"

const VERSION_MISMATCH_ERROR = "pgo client and server version mismatch"

var (
	// Clientset is a client for Kubernetes resources
	Clientset kubeapi.Interface
	// RESTConfig holds the REST configuration for a Kube client
	RESTConfig *rest.Config
)

// MetricsFlag if set to true will cause crunchy-postgres-exporter to be added into new clusters
var MetricsFlag, BadgerFlag bool

// AuditFlag if set to true will cause auditing to occur in the logs
var AuditFlag bool

// DebugFlag is the debug flag value
var DebugFlag bool

// BasicAuth comes from the apiserver config
var BasicAuth bool

// Namespace comes from the apiserver config in this version
var (
	PgoNamespace     string
	InstallationName string
)

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

// NamespaceOperatingMode defines the namespace operating mode for the cluster,
// e.g. "dynamic", "readonly" or "disabled".  See type NamespaceOperatingMode
// for detailed explanations of each mode available.
var namespaceOperatingMode ns.NamespaceOperatingMode

func Initialize() {
	PgoNamespace = os.Getenv("PGO_OPERATOR_NAMESPACE")
	if PgoNamespace == "" {
		log.Info("PGO_OPERATOR_NAMESPACE environment variable is not set and is required, this is the namespace that the Operator is to run within.")
		os.Exit(2)
	}
	log.Info("Pgo Namespace is [" + PgoNamespace + "]")

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

	connectToKube()

	initializePerms()

	err := Pgo.GetConfig(Clientset, PgoNamespace)
	if err != nil {
		log.Error(err)
		log.Error("error in Pgo configuration")
		os.Exit(2)
	}

	initConfig()

	// look through all the pgouser secrets in the Operator's
	// namespace and set a generated password for any that currently
	// have an empty password set
	setRandomPgouserPasswords()

	if err := setNamespaceOperatingMode(); err != nil {
		log.Error(err)
		os.Exit(2)
	}

	_, err = ns.GetInitialNamespaceList(Clientset, NamespaceOperatingMode(),
		InstallationName, PgoNamespace)
	if err != nil {
		log.Error(err)
		os.Exit(2)
	}

	log.Infof("Namespace operating mode is '%s'", NamespaceOperatingMode())
}

func connectToKube() {
	client, err := kubeapi.NewClient()
	if err != nil {
		panic(err)
	}

	Clientset = client
	RESTConfig = client.Config
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
}

func BasicAuthCheck(username, password string) bool {
	ctx := context.TODO()

	if !BasicAuth {
		return true
	}

	// see if there is a pgouser Secret for this username
	secretName := "pgouser-" + username
	secret, err := Clientset.CoreV1().Secrets(PgoNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("could not get pgouser secret %s: %s", username, err.Error())
		return false
	}

	return password == string(secret.Data["password"])
}

func BasicAuthzCheck(username, perm string) bool {
	ctx := context.TODO()
	secretName := "pgouser-" + username
	secret, err := Clientset.CoreV1().Secrets(PgoNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		log.Errorf("could not get pgouser secret %s: %s", username, err.Error())
		return false
	}

	// get the roles for this user
	rolesString := string(secret.Data["roles"])
	roles := strings.Split(rolesString, ",")
	if len(roles) == 0 {
		log.Errorf("%s user has no roles ", username)
		return false
	}

	// venture thru each role this user has looking for a perm match
	for _, r := range roles {

		// get the pgorole
		roleSecretName := "pgorole-" + r
		rolesecret, err := Clientset.CoreV1().Secrets(PgoNamespace).Get(ctx, roleSecretName, metav1.GetOptions{})
		if err != nil {
			log.Errorf("could not get pgorole secret %s: %s", r, err.Error())
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

// GetNamespace determines if a user has permission for
// a namespace they are requesting
// a valid requested namespace is required
func GetNamespace(clientset kubernetes.Interface, username, requestedNS string) (string, error) {
	log.Debugf("GetNamespace username [%s] ns [%s]", username, requestedNS)

	if requestedNS == "" {
		return requestedNS, errors.New("empty namespace is not valid from pgo clients")
	}

	iAccess, uAccess, err := UserIsPermittedInNamespace(username, requestedNS)
	if err != nil {
		return requestedNS, fmt.Errorf("Error when determining whether user [%s] is allowed access to "+
			"namespace [%s]: %s", username, requestedNS, err.Error())
	}
	if !iAccess {
		errMsg := fmt.Sprintf("namespace [%s] is not part of the Operator installation", requestedNS)
		return requestedNS, errors.New(errMsg)
	}
	if !uAccess {
		errMsg := fmt.Sprintf("user [%s] is not allowed access to namespace [%s]", username, requestedNS)
		return requestedNS, errors.New(errMsg)
	}

	return requestedNS, nil
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

func IsValidStorageName(name string) bool {
	_, ok := Pgo.Storage[name]
	return ok
}

// ValidateNodeLabel
// returns error if node label is invalid based on format
func ValidateNodeLabel(nodeLabel string) error {
	parts := strings.Split(nodeLabel, "=")
	if len(parts) != 2 {
		return errors.New(nodeLabel + " node label does not follow key=value format")
	}

	return nil
}

// UserIsPermittedInNamespace returns installation access and user access.
// Installation access means a namespace belongs to this Operator installation.
// User access means this user has access to a namespace.
func UserIsPermittedInNamespace(username, requestedNS string) (bool, bool, error) {
	ctx := context.TODO()
	var iAccess, uAccess bool

	if err := ns.ValidateNamespacesWatched(Clientset, NamespaceOperatingMode(), InstallationName,
		requestedNS); err != nil {
		if !errors.Is(err, ns.ErrNamespaceNotWatched) {
			return false, false, err
		}
	} else {
		iAccess = true
	}

	if iAccess {
		// get the pgouser Secret for this username
		userSecretName := "pgouser-" + username
		userSecret, err := Clientset.CoreV1().Secrets(PgoNamespace).Get(ctx, userSecretName, metav1.GetOptions{})
		if err != nil {
			log.Errorf("could not get pgouser secret %s: %s", username, err.Error())
			return false, false, err
		}

		// handle the case of a user in pgouser with "" (all) namespaces, otherwise check the
		// namespaces config in the user secret
		nsstring := string(userSecret.Data["namespaces"])
		if nsstring == "" {
			uAccess = true
		} else {
			nsList := strings.Split(nsstring, ",")
			for _, v := range nsList {
				ns := strings.TrimSpace(v)
				if ns == requestedNS {
					uAccess = true
				}
			}
		}
	}

	return iAccess, uAccess, nil
}

// WriteTLSCert is a legacy method that writes the server certificate and key to
// files from the PGOSecretName secret or generates a new key (writing to both
// the secret and the expected files
func WriteTLSCert(certPath, keyPath string) error {
	ctx := context.TODO()
	pgoSecret, err := Clientset.CoreV1().Secrets(PgoNamespace).Get(ctx, PGOSecretName, metav1.GetOptions{})
	// if the TLS certificate secret is not found, attempt to generate one
	if err != nil {
		log.Infof("%s Secret NOT found in namespace %s", PGOSecretName, PgoNamespace)

		if err := generateTLSCert(certPath, keyPath); err != nil {
			log.Error("error generating pgo.tls Secret")
			return err
		}

		return nil
	}

	// otherwise, write the TLS sertificate to the certificate and key path
	log.Infof("%s Secret found in namespace %s", PGOSecretName, PgoNamespace)
	log.Infof("cert key data len is %d", len(pgoSecret.Data[corev1.TLSCertKey]))

	if err := ioutil.WriteFile(certPath, pgoSecret.Data[corev1.TLSCertKey], 0o600); err != nil {
		return err
	}

	log.Infof("private key data len is %d", len(pgoSecret.Data[corev1.TLSPrivateKeyKey]))

	if err := ioutil.WriteFile(keyPath, pgoSecret.Data[corev1.TLSPrivateKeyKey], 0o600); err != nil {
		return err
	}

	return nil
}

// generateTLSCert generates a self signed cert and stores it in both
// the PGOSecretName Secret and certPath, keyPath files
func generateTLSCert(certPath, keyPath string) error {
	ctx := context.TODO()
	var err error

	// generate private key
	var privateKey *ecdsa.PrivateKey
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
	newSecret := corev1.Secret{}
	newSecret.Name = PGOSecretName
	newSecret.ObjectMeta.Labels = make(map[string]string)
	newSecret.ObjectMeta.Labels[config.LABEL_VENDOR] = "crunchydata"
	newSecret.Data = make(map[string][]byte)
	newSecret.Data[corev1.TLSCertKey] = caCertBytes
	newSecret.Data[corev1.TLSPrivateKeyKey] = privateKeyBytes
	newSecret.Type = corev1.SecretTypeTLS

	_, err = Clientset.CoreV1().Secrets(PgoNamespace).Create(ctx, &newSecret, metav1.CreateOptions{})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	if err := ioutil.WriteFile(certPath, newSecret.Data[corev1.TLSCertKey], 0o600); err != nil {
		return err
	}
	if err := ioutil.WriteFile(keyPath, newSecret.Data[corev1.TLSPrivateKeyKey], 0o600); err != nil {
		return err
	}

	return err
}

// setNamespaceOperatingMode set the namespace operating mode for the Operator by calling the
// proper utility function to determine which mode is applicable based on the current
// permissions assigned to the Operator Service Account.
func setNamespaceOperatingMode() error {
	nsOpMode, err := ns.GetNamespaceOperatingMode(Clientset)
	if err != nil {
		return err
	}
	namespaceOperatingMode = nsOpMode

	return nil
}

// setRandomPgouserPasswords looks through the pgouser secrets in the Operator's
// namespace. If any have an empty password, it generates a random password,
// Base64 encodes it, then stores it in the relevant PGO user's secret
func setRandomPgouserPasswords() {
	ctx := context.TODO()

	selector := "pgo-pgouser=true,vendor=crunchydata"
	secrets, err := Clientset.CoreV1().Secrets(PgoNamespace).
		List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Warnf("Could not get pgouser secrets in namespace: %s", PgoNamespace)
		return
	}

	for _, secret := range secrets.Items {
		// check if password is set. if it is, continue.
		if len(secret.Data["password"]) > 0 {
			continue
		}

		log.Infof("Password in pgouser secret %s for operator installation %s in namespace %s is empty. "+
			"Setting a generated password.", secret.Name, InstallationName, PgoNamespace)

		// generate the password using the default password length
		generatedPassword, err := util.GeneratePassword(util.DefaultGeneratedPasswordLength)
		if err != nil {
			log.Errorf("Could not generate password for pgouser secret %s for operator installation %s in "+
				"namespace %s", secret.Name, InstallationName, PgoNamespace)
			continue
		}

		// create the password patch
		patch, err := kubeapi.NewMergePatch().Add("stringData", "password")(generatedPassword).Bytes()
		if err != nil {
			log.Errorf("Could not generate password patch for pgouser secret %s for operator installation "+
				"%s in namespace %s", secret.Name, InstallationName, PgoNamespace)
			continue
		}

		// patch the pgouser secret with the new password
		if _, err := Clientset.CoreV1().Secrets(PgoNamespace).Patch(ctx, secret.Name, types.MergePatchType,
			patch, metav1.PatchOptions{}); err != nil {
			log.Errorf("Could not patch pgouser secret %s with generated password for operator installation "+
				"%s in namespace %s", secret.Name, InstallationName, PgoNamespace)
		}
	}
}

// NamespaceOperatingMode returns the namespace operating mode for the current Operator
// installation, which is stored in the "namespaceOperatingMode" variable
func NamespaceOperatingMode() ns.NamespaceOperatingMode {
	return namespaceOperatingMode
}
