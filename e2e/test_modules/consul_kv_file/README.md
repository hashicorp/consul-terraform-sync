# Consul KV File Module
Module for end-to-end testing of Consul KV condition triggers.

## Features
This module creates a file for each Consul KV pair set in the `consul_kv` variable, where the filename is the key and the file content is the value. It additionally creates a file for each service set in the `services` variable, where the filename is the service name and the content is the address of the service.
