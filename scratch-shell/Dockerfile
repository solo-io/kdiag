FROM --platform=${TARGETPLATFORM} docker.io/library/ubuntu:22.04 as builder

# Install dependencies
# note: we can't use linux-headers-$(uname -r), as we don't know the host kernel will match.
# we just need some version of the headers so we can build busybox.
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    lsb-release \
    gdb gcc libc6-dev \
    make \
    linux-libc-dev linux-headers-5.15.0-27-generic \
    && rm -rf /var/lib/apt/lists/*

# RUN ln -s /usr/include/asm-generic /usr/include/asm
RUN ln -s /usr/include/*-linux-gnu/asm/ /usr/include/asm

COPY .config build.sh enter.c /scratch-shell/
RUN cd /scratch-shell && ./build.sh && \
    cp ./built/ash /usr/local/bin && \
    cp ./built/enter /usr/local/bin

FROM --platform=${TARGETPLATFORM} docker.io/library/ubuntu:22.04

COPY --from=builder /usr/local/bin/ash /usr/local/bin/ash
COPY --from=builder /usr/local/bin/enter /usr/local/bin/enter