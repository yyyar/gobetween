# Guide to make gobetween Docker image for Alpine

1. Firstly, we need to build gobetween for Alpine environment
   - `git clone https://github.com/yyyar/gobetween.git`
   - `cd ./gobetween/docker/alpine`
   - Create new image: `docker build -f ./alpine-build -t gobetween-alpine-compile .` (notice the last dot `.`)
   - Run a container: `docker run -v ../../:/go/src/app gobetween-alpine-compile`
   - The container will exit when compilation finishes.

3. Build the useable gobetween image:
   - `docker build -t gobetween:0.5.0-alpine3.6 .`
4. Clean up unused image:
   - `docker image rm gobetween-alpine-compile`
5. Now you have a tiny gobetween image ready to use. 
   - Check it out: `docker image ls`