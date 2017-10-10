#!/bin/sh

curl http://169.254.169.254/latest/meta-data/instance-id > /etc/host_hostname

/bin/logspout "$@"