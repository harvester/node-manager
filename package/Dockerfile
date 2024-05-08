# syntax=docker/dockerfile:1.7.0

FROM registry.suse.com/bci/bci-base:15.5

RUN zypper -n rm container-suseconnect && \
    zypper -n clean -a && rm -rf /tmp/* /var/tmp/* /usr/share/doc/packages/*

ARG TARGETPLATFORM

RUN if [ "$TARGETPLATFORM" != "linux/amd64" ]; then \
    echo "Error: Unsupported TARGETPLATFORM: $TARGETPLATFORM" && \
    exit 1; \
    fi

ENV ARCH=${TARGETPLATFORM#linux/}

COPY bin/harvester-node-manager-${ARCH} /usr/bin/
CMD ["harvester-node-manager"]
