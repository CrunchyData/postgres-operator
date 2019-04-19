package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"github.com/crunchydata/postgres-operator/kubeapi"
	"github.com/crunchydata/postgres-operator/tlsutil"
	corev1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"math"
	"math/big"
	"os"
	"time"
)

var (
	kubeconfig = flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
)

const (
	rsaKeySize   = 2048
	duration365d = time.Hour * 24 * 365

	// TLSCACertKey is the key for tls CA certificates.
	TLSCACertKey = tlsutil.TLSCACertKey

	// TLSCertKey is the key for tls certificates.
	TLSCertKey = corev1.TLSCertKey

	//name of pgo.tls Secret
	PGOSecretName = "pgo.tls"
)

func main() {
	flag.Parse()
	// uses the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	namespace := "pgouser1"
	lo := meta_v1.ListOptions{}
	pods, err := clientset.CoreV1().Pods(namespace).List(lo)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
	fmt.Printf("There are %d pods in namespace %s\n", len(pods.Items), namespace)
	var ct tlsutil.CertType
	ct = 8
	fmt.Printf("certtype %d", ct)

	var pgoSecret *corev1.Secret
	var found bool
	namespace = "pgo"
	pgoSecret, found, err = kubeapi.GetSecret(clientset, PGOSecretName, namespace)
	if found {
		fmt.Printf("%s Secret found in namespace %s\n", PGOSecretName, namespace)
		fmt.Printf("%s Secret Name \n", pgoSecret.Name)
	} else {
		fmt.Printf("%s Secret NOT found in namespace %s\n", PGOSecretName, namespace)
	}
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	//generate private key
	var privateKey *rsa.PrivateKey
	privateKey, err = newPrivateKey()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	privateKeyBytes := encodePrivateKeyPEM(privateKey)
	fmt.Printf("privateKeyBytes len %d\n", len(privateKeyBytes))

	var caCert *x509.Certificate
	caCert, err = newSelfSignedCACertificate(privateKey)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	caCertBytes := encodeCertificatePEM(caCert)
	fmt.Printf("caCertBytes len %d\n", len(caCertBytes))

	// CreateSecret
	newSecret := corev1.Secret{}
	newSecret.Name = "jeff.tls"
	newSecret.ObjectMeta.Labels = make(map[string]string)
	newSecret.ObjectMeta.Labels["vendor"] = "crunchydata"
	newSecret.Data = make(map[string][]byte)
	newSecret.Data[corev1.TLSCertKey] = caCertBytes
	newSecret.Data[corev1.TLSPrivateKeyKey] = privateKeyBytes
	newSecret.Type = corev1.SecretTypeTLS

	err = kubeapi.CreateSecret(clientset, &newSecret, namespace)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

}

// newPrivateKey returns randomly generated RSA private key.
func newPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

// encodePrivateKeyPEM encodes the given private key pem and returns bytes (base64).
func encodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// encodeCertificatePEM encodes the given certificate pem and returns bytes (base64).
func encodeCertificatePEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

// parsePEMEncodedCert parses a certificate from the given pemdata
func parsePEMEncodedCert(pemdata []byte) (*x509.Certificate, error) {
	decoded, _ := pem.Decode(pemdata)
	if decoded == nil {
		return nil, errors.New("no PEM data found")
	}
	return x509.ParseCertificate(decoded.Bytes)
}

// parsePEMEncodedPrivateKey parses a private key from given pemdata
func parsePEMEncodedPrivateKey(pemdata []byte) (*rsa.PrivateKey, error) {
	decoded, _ := pem.Decode(pemdata)
	if decoded == nil {
		return nil, errors.New("no PEM data found")
	}
	return x509.ParsePKCS1PrivateKey(decoded.Bytes)
}

// newSelfSignedCACertificate returns a self-signed CA certificate based on given configuration and private key.
// The certificate has one-year lease.
func newSelfSignedCACertificate(key *rsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber:          serial,
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(duration365d).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// newSignedCertificate signs a certificate using the given private key, CA and returns a signed certificate.
// The certificate could be used for both client and server auth.
// The certificate has one-year lease.
func newSignedCertificate(cfg *tlsutil.CertConfig, dnsNames []string, key *rsa.PrivateKey, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	eku := []x509.ExtKeyUsage{}
	switch cfg.CertType {
	case tlsutil.ClientCert:
		eku = append(eku, x509.ExtKeyUsageClientAuth)
	case tlsutil.ServingCert:
		eku = append(eku, x509.ExtKeyUsageServerAuth)
	case tlsutil.ClientAndServingCert:
		eku = append(eku, x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth)
	}
	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     dnsNames,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(duration365d).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  eku,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}
