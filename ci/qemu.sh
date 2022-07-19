#!/bin/bash
# this is short version of
# https://github.com/qemu/qemu/blob/master/scripts/qemu-binfmt-conf.sh

# if you have trouble building multi-arch docker images,
# download qemu-aarch64-static and run this:

# install qemu with dnf install qemu-user-static
# or dnf qemu-user-static-aarch64 qemu-user-static-arm

# as this is by design a static binary, you can also download a binary from here:
# https://github.com/multiarch/qemu-user-static/releases

cpu=aarch64

aarch64_magic='\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\xb7\x00'
aarch64_mask='\xff\xff\xff\xff\xff\xff\xff\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff'
aarch64_family=arm

# persistent flag
flags=F
magic=$(eval echo \$${cpu}_magic)
mask=$(eval echo \$${cpu}_mask)
family=$(eval echo \$${cpu}_family)

qemu=$(which qemu-aarch64-static)
# use qemu env var if provided, or fallback to use qemu-aarch64-static in path.
QEMU=${QEMU:-$qemu}

echo ":qemu-$cpu:M::$magic:$mask:$QEMU:$flags" > /proc/sys/fs/binfmt_misc/register