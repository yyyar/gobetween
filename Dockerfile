FROM scratch
COPY bin/gobetween  /usr/bin/
CMD ["/usr/bin/gobetween", "-c", "/etc/gobetween/conf/gobetween.toml"]

LABEL org.label-schema.vendor="Gobetween" \
      org.label-schema.url="https://gobetween.io" \
      org.label-schema.name="Gobetween" \
      org.label-schema.description="Modern & minimalistic load balancer for the Сloud era" 
