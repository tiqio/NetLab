FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        curl \
        dnsutils \
        ethtool \
        iproute2 \
        iputils-ping \
        iputils-tracepath \
        isc-dhcp-client \
        netcat-openbsd \
        procps \
        tcpdump \
        traceroute \
    && rm -rf /var/lib/apt/lists/*

CMD ["sleep", "infinity"]
