package rmdata

import (
	"github.com/crunchydata/postgres-operator/config"
	"github.com/crunchydata/postgres-operator/kubeapi"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

func Delete(request Request) {
	log.Infof("rmdata.Process %v", request)

	if request.RemoveData {
	}

	removeClusterJobs(request)

	removeCluster(request)

	if request.RemoveBackup {
		removeBackups(request)
	}

}

func removeBackups(request Request) {

	//see if a pgbasebackup PVC exists
	backupPVCName := request.RemoveCluster + "-backup"
	log.Infof("pgbasebackup backup pvc: %s", backupPVCName)
	pvc, found, err := kubeapi.GetPVC(request.Clientset, request.RemoveCluster, request.RemoveNamespace)
	if found {
		log.Infof("pgbasebackup backup pvc: found")
		err = kubeapi.DeletePVC(request.Clientset, pvc.Name, request.RemoveNamespace)
		if err != nil {
			log.Errorf("error removing pgbasebackup pvc %s %s", backupPVCName, err.Error())
		} else {
			log.Infof("removed pgbasebackup pvc %s", backupPVCName)
		}
	} else {
		log.Infof("pgbasebackup backup pvc: NOT found")
	}

	//delete pgbackrest PVC if it exists

	selector := config.LABEL_PG_CLUSTER + "=" + request.RemoveCluster
	log.Infof("remove backrest pvc selector [%s]", selector)

	var pvcList *v1.PersistentVolumeClaimList
	pvcList, err = kubeapi.GetPVCs(request.Clientset, selector, request.RemoveNamespace)
	if len(pvcList.Items) > 0 {
		for _, v := range pvcList.Items {
			err = kubeapi.DeletePVC(request.Clientset, v.Name, request.RemoveNamespace)
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
	selector := config.LABEL_PG_CLUSTER + "=" + request.RemoveCluster + "," + config.LABEL_SERVICE_NAME + "=" + request.RemoveCluster + "-replica"
	pods, err := kubeapi.GetPods(request.Clientset, selector, request.RemoveNamespace)
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
			command = append(command, "/pgdata/"+v.ObjectMeta.Labels[config.LABEL_REPLICA_NAME])
			stdout, stderr, err := kubeapi.ExecToPodThroughAPI(request.RESTConfig, request.Clientset, command, v.Spec.Containers[0].Name, v.Name, request.RemoveNamespace, nil)
			if err != nil {
				log.Errorf("error execing into remove data pod %s command %s error %s", v.Name, command, err.Error())
			}
			log.Infof("stdout=[%s] stderr=[%s]", stdout, stderr)
		}
	}

	//get the primary

	//primaries should have the label of
	//the form deployment-name=somedeploymentname and service-name=somecluster
	selector = config.LABEL_PG_CLUSTER + "=" + request.RemoveCluster + "," + config.LABEL_SERVICE_NAME + "=" + request.RemoveCluster
	pods, err = kubeapi.GetPods(request.Clientset, selector, request.RemoveNamespace)
	if err != nil {
		log.Errorf("error selecting primary pod %s %s", selector, err.Error())
	}

	if len(pods.Items) > 0 {
	}

}

func removeClusterJobs(request Request) {
}

func removeCluster(request Request) {
}
