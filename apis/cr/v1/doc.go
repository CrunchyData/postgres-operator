/*
Crunchy PostgreSQL Operator API

The Crunchy PostgreSQL Operator API defines HTTP(S) interactions with the Crunchy PostgreSQL Operator.


## Direct API Calls

The API can also be accessed by interacting directly with the API server. This
can be done by making HTTP requests with curl to get information from the
server. In order to make these calls you will need to provide certificates along
with your request using the `--cacert`, `--key`, and `--cert` flags. Next you
will need to provide the username and password for the RBAC along with a header
that includes the content type and the `--insecure` flag. These flags will be
the same for all of your interactions with the API server and can be seen in the
following examples.


###### Get API Server Version

The most basic example of this interaction is getting the version of the API
server. You can send a GET request to `$PGO_APISERVER_URL/version` and this will
send back a json response including the API server version. You must specify the
client version that matches the API server version as part of the request.

The API server is setup to work with the pgo command line interface so the
parameters that are passed to the server can be found by looking at the related
flags. For example, the series parameter used in the `create` example below is
the same as the `-e, --series` flag that is described in the [pgo cli docs](https://access.crunchydata.com/documentation/postgres-operator/4.1.0/operatorcli/cli/pgo_create_cluster/).
```
curl --cacert $PGO_CA_CERT --key $PGO_CLIENT_KEY --cert $PGO_CA_CERT -u \
pgoadmin:examplepassword -H "Content-Type:application/json" --insecure -X \
GET $PGO_APISERVER_URL/version
```

#### Body examples
In the following examples data is being passed to the apiserver using a json
structure. These json structures are defined in the following documentation.

```
curl --cacert $PGO_CA_CERT --key $PGO_CLIENT_KEY --cert $PGO_CA_CERT -u \
pgoadmin:examplepassword -H "Content-Type:application/json" --insecure -X GET \
"$PGO_APISERVER_URL/workflow/<id>?version=<client-version>&namespace=<namespace>"
```

###### Create Cluster
You can create a cluster by sending a POST request to
`$PGO_APISERVER_URL/clusters`. In this example `--data` is being sent to the
API URL that includes the client version that was returned from the version
call, the namespace where the cluster should be created, the name of the new
cluster and the series number. Series sets the number of clusters that will be
created in the namespace.

```
curl --cacert $PGO_CA_CERT --key $PGO_CLIENT_KEY --cert $PGO_CA_CERT -u \
pgoadmin:examplepassword -H "Content-Type:application/json" --insecure -X \
POST --data \
    '{"ClientVersion":"4.1.0",
    "Namespace":"pgouser1",
    "Name":"mycluster",
    "Series":1}' \
$PGO_APISERVER_URL/clusters
```



###### Show and Delete Cluster
The last two examples show you how to `show` and `delete` a cluster. Notice
how instead of passing `"Name":"mycluster"` you pass `"Clustername":"mycluster"
to reference a cluster that has already been created. For the show cluster
example you can replace `"Clustername":"mycluster"` with `"AllFlag":true` to
show all of the clusters that are in the given namespace.

```
curl --cacert $PGO_CA_CERT --key $PGO_CLIENT_KEY --cert $PGO_CA_CERT -u \
pgoadmin:examplepassword -H "Content-Type:application/json" --insecure -X \
POST --data \
  '{"ClientVersion":"4.1.0",
  "Namespace":"pgouser1",
  "Clustername":"mycluster"}' \
$PGO_APISERVER_URL/showclusters
```

```
curl --cacert $PGO_CA_CERT --key $PGO_CLIENT_KEY --cert $PGO_CA_CERT -u \
pgoadmin:examplepassword -H "Content-Type:application/json" --insecure -X \
POST --data \
  '{"ClientVersion":"4.1.0",
  "Namespace":"pgouser1",
  "Clustername":"mycluster"}' \
$PGO_APISERVER_URL/clustersdelete
```

  Schemes: http, https
  BasePath: /
  Version: 4.1.0
  License: Apache 2.0 http://www.apache.org/licenses/LICENSE-2.0
  Contact: Crunchy Data<info@crunchydata.com> https://www.crunchydata.com/


	Consumes:
	- application/json

	Produces:
	- application/json

swagger:meta
*/
package v1

// +k8s:deepcopy-gen=package,register

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
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
