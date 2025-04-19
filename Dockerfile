###########################################################################

FROM alpine
MAINTAINER burble <simon@burble.com>
VOLUME /registry

###########################################################################

RUN apk add --update git tini && rm -rf /var/cache/apk/*

ADD dn42regsrv /usr/local/bin/dn42regsrv
ADD StaticRoot /StaticRoot
RUN mkdir -p /registry && \
    chown -R 1000:1000 /registry /usr/local/bin/dn42regsrv /StaticRoot && \
    chmod u+rx /usr/local/bin/dn42regsrv && \
    chmod -R u+rX /StaticRoot /registry

###########################################################################

USER 1000
WORKDIR /registry
EXPOSE 8042

ENTRYPOINT [ "/sbin/tini", "--" ]
CMD [ "/usr/local/bin/dn42regsrv", "-d", "/registry", "-s", "/StaticRoot" ]

###########################################################################
# end of file