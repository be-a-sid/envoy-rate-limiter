FROM envoyproxy/envoy:contrib-v1.29.3
RUN apt-get update
COPY ./envoy/envoy.yaml /etc/envoy.yaml
COPY ./rate-limiter-plugin/build/rate-limiter.so /lib/rate-limiter.so
CMD /usr/local/bin/envoy -c /etc/envoy.yaml