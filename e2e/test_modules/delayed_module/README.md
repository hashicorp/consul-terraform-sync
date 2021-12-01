# Delayed Module
Module for end-to-end testing where the task needs to take a certain amount of time.

## Features
This module sleeps for a given amount of time based on the variable `delay`, where the default is `5s`. It also creates a file for each service set in the `services` variable and a file that has `Hello, <Consul service name>!`for each Consul service.
