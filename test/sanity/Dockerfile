FROM mcr.microsoft.com/aks/fundamental/base-ubuntu:v0.0.5

WORKDIR /

RUN apt-get update && apt-get install -y util-linux e2fsprogs mount ca-certificates udev xfsprogs cifs-utils procps

COPY bin/csi-sanity /usr/bin

COPY azurediskplugin run-sanity-test.sh /

CMD ./run-sanity-test.sh
