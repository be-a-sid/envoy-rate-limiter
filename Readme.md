
### Steps to start Envoy
```
docker build -t envoy-rate-limiter  -f ./envoy/Dockerfile .

docker container run -d --name erl -p 10210:10210 -p 9901:9901 envoy-rate-limiter

docker container logs -f erl

docker container stop erl

docker container remove erl
```