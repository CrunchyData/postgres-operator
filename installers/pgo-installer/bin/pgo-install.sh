#!/bin/bash

/usr/bin/env ansible-playbook -i /ansible-inventory/inventory --tags=$1 /ansible/main.yml
