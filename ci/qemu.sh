#!/bin/bash
# this is short version of
# https://github.com/qemu/qemu/blob/master/scripts/qemu-binfmt-conf.sh

# if you have trouble building multi-arch docker images,
# download qemu-aarch64-static and run this:

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

echo ":qemu-$cpu:M::$magic:$mask:$qemu:$flags" > /proc/sys/fs/binfmt_misc/register