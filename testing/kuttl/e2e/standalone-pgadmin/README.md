** pgAdmin **

Note: due to the (random) namespace being part of the host, we cannot check the configmap using the usual assert/file pattern.

*Phase one*

* 00:
  * create a pgadmin with no server groups;
  * check the correct existence of the secret, configmap, and pod.
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

pgAdmin v7 vs v8 Notes: 
pgAdmin v8 includes updates to `setup.py` which alter how the `dump-servers` argument
is called:
- v7: https://github.com/pgadmin-org/pgadmin4/blob/REL-7_8/web/setup.py#L175
- v8: https://github.com/pgadmin-org/pgadmin4/blob/REL-8_5/web/setup.py#L79

You will also notice a difference in the `assert.yaml` files between the stored
config and the config returned by the `dump-servers` command. The additional setting,
`"TunnelPort": "22"`, is due to the new defaulting behavior added to pgAdmin for psycopg3.
See
- https://github.com/pgadmin-org/pgadmin4/commit/5e0daccf7655384db076512247733d7e73025d1b
- https://github.com/pgadmin-org/pgadmin4/blob/REL-8_5/web/pgadmin/utils/driver/psycopg3/server_manager.py#L94
