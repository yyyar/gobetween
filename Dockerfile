FROM centos:7

RUN mkdir -p /etc/gobetween/conf

COPY bin/gobetween  /usr/bin/

CMD ["/usr/bin/gobetween", "-c", "/etc/gobetween/conf/gobetween.toml"]
