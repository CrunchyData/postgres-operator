package rmdata

import (
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"time"
)

func Delete(request Request) {
	log.Infof("rmdata.Process %v", request)

	if request.RemoveData {
		pvcList, err := getPVCs(request)
		if err != nil {
			log.Error(err)
		}

		if request.IsBackup {
		} else if request.IsReplica {
			removeOnlyReplicaData(request)
			removeServices(request)
		} else {
			//remove pgdata
			removeData(request)

			//remove secrets only if this is an entire cluster being
			//removed
			if request.IsReplica == false && request.IsBackup == false {
				removeUserSecrets(request)
			}

			removeClusterJobs(request)

			tries := 0
			maxtries := 4
			found := false
			for i := 0; i < maxtries; i++ {
				found = false
				deployments, err := kubeapi.GetDeployments(request.Clientset,
					config.LABEL_PG_CLUSTER+"="+request.ClusterName, request.Namespace)
				if err != nil {
					log.Error(err)
				}
				if len(deployments.Items) > 0 {
					removeCluster(request)
					tries++
					found = true
				}
				if found {
					log.Info("sleeping to wait for Deployments to fully terminate")
					time.Sleep(time.Second * time.Duration(4))
				}
			}

			removeServices(request)

			if request.RemoveBackup {
				removeBackrestRepo(request)
			}

			removeAddons(request)

			removePgreplicas(request)

			removePgtasks(request)

			if request.RemoveBackup {
				removeBackups(request)
			}

			removePVCs(pvcList, request)

		}
	}

}

func removeBackups(request Request) {

	//see if a pgbasebackup PVC exists
	backupPVCName := request.ClusterName + "-backup"
	log.Infof("pgbasebackup backup pvc: %s", backupPVCName)
	pvc, found, err := kubeapi.GetPVC(request.Clientset, request.ClusterName, request.Namespace)
	if found {
		log.Infof("pgbasebackup backup pvc: found")
		err = kubeapi.DeletePVC(request.Clientset, pvc.Name, request.Namespace)
		if err != nil {
			log.Errorf("error removing pgbasebackup pvc %s %s", backupPVCName, err.Error())
		} else {
			log.Infof("removed pgbasebackup pvc %s", backupPVCName)
		}
	} else {
		log.Infof("pgbasebackup backup pvc: NOT found")
	}

	//delete pgbackrest PVC if it exists

	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName
	log.Infof("remove backrest pvc selector [%s]", selector)

	var pvcList *v1.PersistentVolumeClaimList
	pvcList, err = kubeapi.GetPVCs(request.Clientset, selector, request.Namespace)
	if len(pvcList.Items) > 0 {
		for _, v := range pvcList.Items {
			err = kubeapi.DeletePVC(request.Clientset, v.Name, request.Namespace)
			if err != nil {
				log.Errorf("error removing backrest pvc %s %s", v.Name, err.Error())
			} else {
				log.Infof("removed backrest pvc %s", v.Name)
			}
		}
	}

}

func removeData(request Request) {
	//get the replicas
	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName + "," + config.LABEL_SERVICE_NAME + "=" + request.ClusterName + "-replica"
	log.Debugf("removeData selector %s", selector)
	pods, err := kubeapi.GetPods(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Errorf("error selecting replica pods %s %s", selector, err.Error())
	}

	//replicas should have a label on their pod of the
	//form deployment-name=somedeploymentname

	log.Debugf("removeData %d replica pods", len(pods.Items))
	if len(pods.Items) > 0 {
		for _, v := range pods.Items {
			command := make([]string, 0)
			command = append(command, "rm")
			command = append(command, "-rf")
			command = append(command, "/pgdata/"+v.ObjectMeta.Labels[config.LABEL_REPLICA_NAME])
			stdout, stderr, err := kubeapi.ExecToPodThroughAPI(request.RESTConfig, request.Clientset, command, v.Spec.Containers[0].Name, v.Name, request.Namespace, nil)
			if err != nil {
				log.Errorf("error execing into remove data pod %s command %s error %s", v.Name, command, err.Error())
			}
			log.Infof("removeData replica stdout=[%s] stderr=[%s]", stdout, stderr)
		}
	}

	//get the primary

	//primaries should have the label of
	//the form deployment-name=somedeploymentname and service-name=somecluster
	selector = config.LABEL_PG_CLUSTER + "=" + request.ClusterName + "," + config.LABEL_SERVICE_NAME + "=" + request.ClusterName
	pods, err = kubeapi.GetPods(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Errorf("error selecting primary pod %s %s", selector, err.Error())
	}

	if len(pods.Items) > 0 {
		pod := pods.Items[0]
		command := make([]string, 0)
		command = append(command, "rm")
		command = append(command, "-rf")
		command = append(command, "/pgdata/"+pod.ObjectMeta.Labels[config.LABEL_REPLICA_NAME])
		stdout, stderr, err := kubeapi.ExecToPodThroughAPI(request.RESTConfig, request.Clientset, command, pod.Spec.Containers[0].Name, pod.Name, request.Namespace, nil)
		if err != nil {
			log.Errorf("error execing into remove data pod %s command %s error %s", pod.Name, command, err.Error())
		}
		log.Infof("removeData primary stdout=[%s] stderr=[%s]", stdout, stderr)
	}

}

func removeClusterJobs(request Request) {
	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName
	jobs, err := kubeapi.GetJobs(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}
	for i := 0; i < len(jobs.Items); i++ {
		job := jobs.Items[i]
		err := kubeapi.DeleteJob(request.Clientset, job.Name, request.Namespace)
		if err != nil {
			log.Error(err)
		}
	}
}

func removeCluster(request Request) {

	deployments, err := kubeapi.GetDeployments(request.Clientset,
		config.LABEL_PG_CLUSTER+"="+request.ClusterName, request.Namespace)
	if err != nil {
		log.Error(err)
	}

	for _, d := range deployments.Items {
		err = kubeapi.DeleteDeployment(request.Clientset, d.ObjectMeta.Name, request.Namespace)
		if err != nil {
			log.Error(err)
		}
	}
}

func removeUserSecrets(request Request) {
	//get all that match pg-cluster=db
	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName

	secrets, err := kubeapi.GetSecrets(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}

	for _, s := range secrets.Items {
		if s.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] == "" {
			err := kubeapi.DeleteSecret(request.Clientset, s.ObjectMeta.Name, request.Namespace)
			if err != nil {
				log.Error(err)
			}
		}
	}

}

func removeOnlyReplicaData(request Request) {
	//get the replica pod only, this is the case where
	//a user scales down a replica, in this case the DeploymentName
	//is used to identify the correct pod
	//in this case, the clustername is the replica deployment name
	selector := config.LABEL_DEPLOYMENT_NAME + "=" + request.ClusterName

	pods, err := kubeapi.GetPods(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Errorf("error selecting replica pods %s %s", selector, err.Error())
	}

	//replicas should have a label on their pod of the
	//form deployment-name=somedeploymentname

	if len(pods.Items) > 0 {
		for _, v := range pods.Items {
			command := make([]string, 0)
			command = append(command, "rm")
			command = append(command, "-rf")
			command = append(command, "/pgdata/"+v.ObjectMeta.Labels[config.LABEL_DEPLOYMENT_NAME])
			stdout, stderr, err := kubeapi.ExecToPodThroughAPI(request.RESTConfig, request.Clientset, command, v.Spec.Containers[0].Name, v.Name, request.Namespace, nil)
			if err != nil {
				log.Errorf("error execing into remove data pod %s command %s error %s", v.Name, command, err.Error())
			}
			log.Infof("stdout=[%s] stderr=[%s]", stdout, stderr)
		}
	}

}

func removeAddons(request Request) {
	//remove pgbouncer

	pgbouncerDepName := request.ClusterName + "-pgbouncer"

	kubeapi.DeleteDeployment(request.Clientset, pgbouncerDepName, request.Namespace)

	//delete the service name=<clustename>-pgbouncer

	kubeapi.DeleteService(request.Clientset, pgbouncerDepName, request.Namespace)

	//remove pgpool
	pgpoolDepName := request.ClusterName + "-pgpool"

	kubeapi.DeleteDeployment(request.Clientset, pgpoolDepName, request.Namespace)

	//delete the service name=<clustename>-pgpool

	kubeapi.DeleteService(request.Clientset, pgpoolDepName, request.Namespace)

}

func removeServices(request Request) {

	//remove any service for this cluster

	selector := config.LABEL_PG_CLUSTER + "=" + request.ClusterName

	services, err := kubeapi.GetServices(request.Clientset, selector, request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}

	for i := 0; i < len(services.Items); i++ {
		svc := services.Items[i]
		err := kubeapi.DeleteService(request.Clientset, svc.Name, request.Namespace)
		if err != nil {
			log.Error(err)
		}
	}

}

func removeBackrestRepo(request Request) {

	depName := request.ClusterName + "-backrest-shared-repo"
	log.Debugf("deleting the backrest repo deployment and service %s", depName)

	err := kubeapi.DeleteDeployment(request.Clientset, depName, request.Namespace)
	if err != nil {
		log.Error(err)
	}

	//delete the service for the backrest repo
	err = kubeapi.DeleteService(request.Clientset, depName, request.Namespace)
	if err != nil {
		log.Error(err)
	}

}

func removePgreplicas(request Request) {
	replicaList := crv1.PgreplicaList{}

	//get a list of pgreplicas for this cluster
	err := kubeapi.GetpgreplicasBySelector(request.RESTClient,
		&replicaList, config.LABEL_PG_CLUSTER+"="+request.ClusterName,
		request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("pgreplicas found len is %d\n", len(replicaList.Items))

	for _, r := range replicaList.Items {
		err = kubeapi.Deletepgreplica(request.RESTClient, r.Spec.Name, request.Namespace)
	}

}

func removePgtasks(request Request) {
	taskList := crv1.PgtaskList{}

	//get a list of pgtasks for this cluster
	err := kubeapi.GetpgtasksBySelector(request.RESTClient,
		&taskList, config.LABEL_PG_CLUSTER+"="+request.ClusterName,
		request.Namespace)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("pgtasks to remove is %d\n", len(taskList.Items))

	for _, r := range taskList.Items {
		err = kubeapi.Deletepgtask(request.RESTClient, r.Spec.Name, request.Namespace)
	}

}

//get the pvc's for this cluster leaving out the backrest repo pvc
func getPVCs(request Request) ([]string, error) {
	pvcList := make([]string, 0)
	deployments, err := kubeapi.GetDeployments(request.Clientset,
		config.LABEL_PG_CLUSTER+"="+request.ClusterName, request.Namespace)
	if err != nil {
		log.Error(err)
		return pvcList, err
	}

	for _, d := range deployments.Items {
		if d.ObjectMeta.Labels[config.LABEL_PGO_BACKREST_REPO] == "" {
			pvcList = append(pvcList, d.ObjectMeta.Name)
		}
	}

	return pvcList, nil

}
func removePVCs(pvcList []string, request Request) error {

	for _, p := range pvcList {
		log.Infof("deleting pvc %s", p)
		err := kubeapi.DeletePVC(request.Clientset, p, request.Namespace)
		if err != nil {
			log.Error(err)
		}
	}

	return nil

}
