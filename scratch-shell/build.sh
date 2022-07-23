#!/bin/bash

set -e
set -x

# build standalone static busybox.
# based partly on:
# https://github.com/robxu9/bash-static
# and notes from https://www.openwall.com/lists/musl/2014/08/08/13
# we use busybox ash instead of bash as ash can be built in "standalone" mode where all the 
# other busybox tools are available magically.

target=linux
arch=x86_64

busybox_version="1_35_0"
musl_version="1.2.3"

script_dir=$(cd $(dirname $0); pwd)

if [ ! -d build ]; then
  mkdir build
fi

pushd build

# pre-prepare gpg for verificaiton
if which gpg 2> /dev/null; then
  echo "= preparing gpg"
  GNUPGHOME="$(mktemp -d)"
  export GNUPGHOME
  # public key for musl
  gpg --batch --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 836489290BB6B70F99FFDA0556BCDB593020450F
fi

# download tarballs
echo "= downloading busybox"

busybox_dir="busybox-${busybox_version}"
if [ ! -e "${busybox_dir}" ]; then
  #https://github.com/mirror/busybox
  curl -LO "https://github.com/mirror/busybox/archive/refs/tags/${busybox_version}.tar.gz"

  if [ $(sha256sum ./1_35_0.tar.gz) -neq "7d563cbce35f12f51afff7d93c1f3adde3fbb4dca3b7dcc34773d6bde3440710  ./1_35_0.tar.gz" ]; then
    echo "= invalid checksum"
    exit 1
  fi

  echo "= extracting busybox"
  tar -xzf "${busybox_version}.tar.gz"
fi

working_dir=$(pwd)
musl_install_dir=${working_dir}/musl-install
musl_dir=musl-${musl_version}

if [ "$(grep ID= < /etc/os-release | head -n1)" = "ID=alpine" ]; then
  echo "= skipping installation of musl because this is alpine linux (and it is already installed)"
else

  if [ ! -e "${musl_dir}" ]; then
    echo "= downloading musl"
    curl -LO https://musl.libc.org/releases/musl-${musl_version}.tar.gz

    if [ $(sha256sum musl-${musl_version}.tar.gz) -neq "7d5b0b6062521e4627e099e4c9dc8248d32a30285e959b7eecaa780cf8cfd4a4  musl-1.2.3.tar.gz" ]; then
      echo "= invalid checksum"
      exit 1
    fi

    if which gpg 2> /dev/null; then
      curl -LO https://musl.libc.org/releases/musl-${musl_version}.tar.gz.asc
      gpg --batch --verify musl-${musl_version}.tar.gz.asc musl-${musl_version}.tar.gz
    fi
    echo "= extracting musl"
    tar -xf musl-${musl_version}.tar.gz
  fi

  if [ ! -e "${musl_install_dir}/bin/musl-gcc" ]; then
    echo "= building musl"
    pushd musl-${musl_version}
    ./configure --prefix="${musl_install_dir}"
    make install
    popd # musl-${musl-version}

    # make environment look more like a cross compiler env
    # see https://www.openwall.com/lists/musl/2014/08/08/13
    pushd ${musl_install_dir}/bin
    ln -s $(which ar) musl-ar
    ln -s $(which strip) musl-strip
    popd
    pushd ${musl_install_dir}/include
    ln -s /usr/include/linux
    ln -s /usr/include/asm
    ln -s /usr/include/asm-generic
    ln -s /usr/include/mtd
    popd
  fi

  echo "= setting CC to musl-gcc"
  echo export CC=${musl_install_dir}/bin/musl-gcc
  export CC=${musl_install_dir}/bin/musl-gcc
  export PATH=${musl_install_dir}/bin:$PATH
fi

if [ ! -e "${busybox_dir}/busybox" ]; then
  export CFLAGS="-static"
  echo export CFLAGS="-static"

  echo "= building busybox"
  # .config incldues:
  # use exe applets, static, cross prefix = musl-, standalone shell
  # for menuconfig you may need to install `sudo dnf install make gcc autoconf ncurses-devel`
  cp ${script_dir}/.config ${busybox_dir}

  pushd ${busybox_dir}
  make busybox
  popd # ${busybox_dir}
fi
popd # build

if [ ! -d built ]; then
  mkdir built
fi

cp build/${busybox_dir}/busybox built/ash
$CC $CFLAGS ./enter.c -o built/enter