## Building the plugin

We utilize a Debian Go image to compile the plugin to `.so` format.

- Build the docker image for the plugin
    ```
    docker build -t rate-limiter-plugin -f Dockerfile.plugin-build .
    ```

- Start the docker container
    ```
    docker run -v ./:/app --name rlp -d rate-limiter-plugin tail -f /dev/null 
    ```

- After the container is started, `sh` into it
    ```
    docker exec -it rlp /bin/sh 
    ```

- Build the plugin
    ```
    go build -ldflags="-s -w" -o ./build/rate-limiter.so -buildmode=c-shared ./plugin
    ```
- If the build is successful, we should see a `rate-limiter.so` file under the `build` folder

- Delete the container and image
    ```
    docker container stop rlp 

    docker container rm rlp

    docker image rm rate-limiter-plugin:latest
    ```