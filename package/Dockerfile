FROM registry.suse.com/bci/bci-base:15.5

RUN zypper -n rm container-suseconnect && \
    zypper -n clean -a && rm -rf /tmp/* /var/tmp/* /usr/share/doc/packages/*

COPY bin/harvester-node-manager /usr/bin/
CMD ["harvester-node-manager"]
