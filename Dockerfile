FROM centos:7

RUN yum -y update && yum -y clean all

COPY bin/gobetween  /usr/bin/

CMD ["/usr/bin/gobetween", "-c", "/etc/gobetween/conf/gobetween.toml"]
