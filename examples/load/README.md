

Loading a sample file, sample.json, requires some setup
at the volume level.

### Making Data Loadable 

#### NFS

If you have an NFS file system, you can create a PV
of type nfs, then create a PVC that specifically mounts
that persistent volume, then copy your sample data into
that location.

Lets assume your NFS mount is at /nfsfileshare on host 192.168.0.107, you can
create a PV as follows:


You then copy your sample data into that PV:

	cp sample.json /nfsfileshare

You then create the PVC, csv-pvc, which is referenced by
the sample load configuration.


#### Storage Class - StorageOS

We test with storageos and it offers a RWO storage class
that can be used for volume provisioning.

To accomplish this, download the storageos CLI from their github site.

To prepare a volume with sample load data, you do the following
for storageos:

Create the PVC that will create a blank storageos volume
for us to use:

    kubectl create -f csv-pvc-sc.yaml

Next, locate the storageos Kube IP address:

    kubectl get svc -n storgeos

As root, set up the storageos environment variables required by the
storageos CLI:

    export KUBECONFIG=/etc/kubernetes/admin.conf
    export STORAGEOS_HOST=10.101.214.130
    export STORAGEOS_PASSWORD=storageos STORAGEOS_USERNAME=storageos
    storageos volume ls

Locate the volume path/name from the above volume listing, then use
that path in a mount command:

    storageos volume mount pgouser1/pvc-3c7491f5-5b9e-11e9-a4a4-52.4.1.1262d /mnt

The above command mounts the storageos volume into /mnt on your local
linux host where you can now access it using normal linux commands.

Next, you copy the sample data file into the storageos volume
which is mounted on your linux host at /mnt:

    sudo cp sample.json /mnt

Unmount the volume so that it can be mounted by the load job:

    storageos volume unmount pgouser1/pvc-3c7491f5-5b9e-11e9-a4a4-52.4.1.1262d

At this point, you have a storageos backed PVC, csv-pvc, that has
the sample.json file loaded into it and its ready to be used
by the load job.   One last change is required to mount the storageos
volume within the Operator load job, that is to specify FSGroup, this
will cause a SecurityContext to be added into the load job which allows
access to the RWO storage class volume:

    FSGroup:  26

To conclude, you would load using a storage class using a command similar
to this one:

    pgo load --load-config=sample-json-load-config-sc.yaml  --selector=name=mycluster -n pgouser1

