# pgAdmin User Management tests

*Note: These tests will only work with pgAdmin version v8 and higher*

## Create pgAdmin with users

* Start pgAdmin with a couple users
* Ensure users exist in pgAdmin with correct settings
* Ensure users exist in the `users.json` file in the pgAdmin secret with the correct settings

## Edit pgAdmin users

* Add a user and edit an existing user
* Ensure users exist in pgAdmin with correct settings
* Ensure users exist in the `users.json` file in the pgAdmin secret with the correct settings

## Delete pgAdmin users

* Remove users from pgAdmin spec
* Ensure users still exist in pgAdmin with correct settings
* Ensure users have been removed from the `users.json` file in the pgAdmin secret
