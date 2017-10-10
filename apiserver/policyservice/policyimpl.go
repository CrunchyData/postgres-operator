package policyservice

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"k8s.io/client-go/rest"

	crv1 "github.com/crunchydata/kraken/apis/cr/v1"
	"github.com/crunchydata/kraken/util"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreatePolicy(RestClient *rest.RESTClient, Namespace, policyName, policyURL, policyFile string) error {
	var err error

	log.Debug("create policy called for " + policyName)
	result := crv1.Pgpolicy{}

	// error if it already exists
	err = RestClient.Get().
		Resource(crv1.PgpolicyResourcePlural).
		Namespace(Namespace).
		Name(policyName).
		Do().
		Into(&result)
	if err == nil {
		log.Infoln("pgpolicy " + policyName + " was found so we will not create it")
		return err
	} else if kerrors.IsNotFound(err) {
		log.Debug("pgpolicy " + policyName + " not found so we will create it")
	} else {
		log.Error("error getting pgpolicy " + policyName + err.Error())
		return err
	}

	// Create an instance of our CRD
	spec := crv1.PgpolicySpec{}
	spec.Name = policyName
	spec.Url = policyURL
	spec.Sql = policyFile

	newInstance := &crv1.Pgpolicy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: policyName,
		},
		Spec: spec,
	}

	err = RestClient.Post().
		Resource(crv1.PgpolicyResourcePlural).
		Namespace(Namespace).
		Body(newInstance).
		Do().Into(&result)

	if err != nil {
		log.Error(" in creating Pgpolicy instance" + err.Error())
		return err
	}
	log.Infoln("created Pgpolicy " + policyName)
	return err

}

func ShowPolicy(RestClient *rest.RESTClient, Namespace string, name string) crv1.PgpolicyList {
	policyList := crv1.PgpolicyList{}

	if name == "all" {
		//get a list of all policies
		err := RestClient.Get().
			Resource(crv1.PgpolicyResourcePlural).
			Namespace(Namespace).
			Do().Into(&policyList)
		if err != nil {
			log.Error("error getting list of policies" + err.Error())
			return policyList
		}
	} else {
		policy := crv1.Pgpolicy{}
		err := RestClient.Get().
			Resource(crv1.PgpolicyResourcePlural).
			Namespace(Namespace).
			Name(name).
			Do().Into(&policy)
		if err != nil {
			log.Error("error getting list of policies" + err.Error())
			return policyList
		}
		policyList.Items = make([]crv1.Pgpolicy, 1)
		policyList.Items[0] = policy
	}

	return policyList

}

func DeletePolicy(Namespace string, RestClient *rest.RESTClient, args []string) error {
	var err error
	// Fetch a list of our policy CRDs
	policyList := crv1.PgpolicyList{}
	err = RestClient.Get().Resource(crv1.PgpolicyResourcePlural).Do().Into(&policyList)
	if err != nil {
		log.Error("error getting policy list" + err.Error())
		return err
	}

	//to remove a policy, you just have to remove
	//the pgpolicy object, the operator will do the actual deletes
	for _, arg := range args {
		policyFound := false
		log.Debug("deleting policy " + arg)
		for _, policy := range policyList.Items {
			if arg == "all" || arg == policy.Spec.Name {
				policyFound = true
				err = RestClient.Delete().
					Resource(crv1.PgpolicyResourcePlural).
					Namespace(Namespace).
					Name(policy.Spec.Name).
					Do().
					Error()
				if err != nil {
					log.Error("error deleting pgpolicy " + arg + err.Error())
					return err
				} else {
					log.Infoln("deleted pgpolicy " + policy.Spec.Name)
				}

			}
		}
		if !policyFound {
			log.Infoln("policy " + arg + " not found")
		}
	}
	return err

}

// pgo apply mypolicy --selector=name=mycluster
func ApplyPolicy(username string, Selector string, Clientset *kubernetes.Clientset, DryRun bool, RestClient *rest.RESTClient, Namespace string, args []string) error {
	var err error
	//validate policies
	labels := make(map[string]string)
	for _, p := range args {
		err = util.ValidatePolicy(RestClient, Namespace, p)
		if err != nil {
			return errors.New("policy " + p + " is not found, cancelling request")
		}

		labels[p] = "pgpolicy"
	}
	//get filtered list of Deployments
	sel := Selector + ",!replica"
	log.Debug("selector string=[" + sel + "]")
	lo := meta_v1.ListOptions{LabelSelector: sel}
	deployments, err := Clientset.ExtensionsV1beta1().Deployments(Namespace).List(lo)
	if err != nil {
		log.Error("error getting list of deployments" + err.Error())
		return err
	}

	if DryRun {
		log.Infoln("policy would be applied to the following clusters:")
		for _, d := range deployments.Items {
			log.Infoln("deployment : " + d.ObjectMeta.Name)
		}
		return err
	}
	var newInstance *crv1.Pgpolicylog
	for _, d := range deployments.Items {
		for _, p := range args {
			log.Debug("apply policy " + p + " on deployment " + d.ObjectMeta.Name + " based on selector " + sel)

			newInstance = getPolicylog(username, p, d.ObjectMeta.Name)

			result := crv1.Pgpolicylog{}
			err = RestClient.Get().
				Resource(crv1.PgpolicylogResourcePlural).
				Namespace(Namespace).
				Name(newInstance.ObjectMeta.Name).
				Do().Into(&result)
			if err == nil {
				log.Infoln(p + " already applied to " + d.ObjectMeta.Name)
				break
			} else {
				if kerrors.IsNotFound(err) {
				} else {
					log.Error(err)
					break
				}
			}

			result = crv1.Pgpolicylog{}
			err = RestClient.Post().
				Resource(crv1.PgpolicylogResourcePlural).
				Namespace(Namespace).
				Body(newInstance).
				Do().Into(&result)
			if err != nil {
				log.Error("error in creating Pgpolicylog CRD instance", err.Error())
				return err
			} else {
				log.Infoln("created Pgpolicylog " + result.ObjectMeta.Name)
			}

		}

	}
	return err

}

func getPolicylog(username, policyname, clustername string) *crv1.Pgpolicylog {

	spec := crv1.PgpolicylogSpec{}
	spec.PolicyName = policyname
	spec.Username = username
	spec.ClusterName = clustername

	newInstance := &crv1.Pgpolicylog{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: policyname + clustername,
		},
		Spec: spec,
	}
	return newInstance

}
