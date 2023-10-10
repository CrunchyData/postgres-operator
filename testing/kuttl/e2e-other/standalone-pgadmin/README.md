** pgAdmin **

Note: requires the StandalonePGAdmin FeatureGate enabled.
Note: due to the (random) namespace being part of the host, we cannot check the configmap using the usual assert/file pattern.

*Phase one*

* 00:
  * create a pgadmin with no server groups;
  * check the correct existence of the secret, service, configmap, and pod.
* 01: dump the servers from pgAdmin and check that the list is empty.

*Phase two*

* 02:
  * create a postgrescluster with a label;
  * update the pgadmin with a selector;
  * check the correct existence of the postgrescluster.
* 03: 
  * check that the configmap is updated in the pgadmin pod;
  * dump the servers from pgAdmin and check that the list has the expected server.

*Phase three*

* 04:
  * create a postgrescluster with the same label;
  * check the correct existence of the postgrescluster.
* 05:
  * check that the configmap is updated in the pgadmin pod;
  * dump the servers from pgAdmin and check that the list has the expected 2 servers.

*Phase four*

* 06:
  * create a postgrescluster with the a different label;
  * update the pgadmin with a second serverGroup;
  * check the correct existence of the postgrescluster.
* 07:
  * check that the configmap is updated in the pgadmin pod;
  * dump the servers from pgAdmin and check that the list has the expected 3 servers.

*Phase five*

* 08:
  * delete a postgrescluster;
  * update the pgadmin with a second serverGroup;
  * check the correct existence of the postgrescluster.
* 09:
  * check that the configmap is updated in the pgadmin pod;
  * dump the servers from pgAdmin and check that the list has the expected 2 servers
