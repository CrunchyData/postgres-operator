package v1

import ()

const PGROOT_SECRET_SUFFIX = "-pgroot-secret"
const PGUSER_SECRET_SUFFIX = "-pguser-secret"
const PGMASTER_SECRET_SUFFIX = "-pgmaster-secret"

const STORAGE_EXISTING = "existing"
const STORAGE_CREATE = "create"
const STORAGE_EMPTYDIR = "emptydir"
const STORAGE_DYNAMIC = "dynamic"

type PgStorageSpec struct {
	PvcName             string `json:"pvcname"`
	StorageClass        string `json:"storageclass"`
	PvcAccessMode       string `json:"pvcaccessmode"`
	PvcSize             string `json:"pvcsize"`
	StorageType         string `json:"storagetype"`
	FSGROUP             string `json:"fsgroup"`
	SUPPLEMENTAL_GROUPS string `json:"supplementalgroups"`
}
