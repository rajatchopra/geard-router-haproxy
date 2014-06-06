
geard-router-haproxy 
====================

This repo can be built and used as a plugin to the geard routing system (https://github.com/openshift/geard). 'geard' creates the intermediate routing structure that is picked up by this package when run as a docker container. Two steps to use this router -

	1. Prepare the docker image using the Dockerfile. This will pull in the latest haproxy source code and compile it. It will build the source code in this repo too. Push the image to an upstream repo.
	2. On a machine where you want to run the router, run a container that uses the above image file (see run.example).

Change something?
	As an example, replace the default_pub_keys.pem, or modify haproxy_template.conf and create a new image.


