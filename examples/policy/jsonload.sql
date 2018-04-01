\c userdb;
create table json_collection
(
	json_imported jsonb
) 
WITH (OIDS=FALSE);

grant all on json_collection to testuser;
