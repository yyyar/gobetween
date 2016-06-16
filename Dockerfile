FROM centos:7

ENV GOMAXPROCS 8

COPY bin/gobetween  /usr/bin/

CMD ["/usr/bin/gobetween", "-c", "/etc/gobetween/conf/gobetween.toml"]
