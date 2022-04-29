#!/bin/bash

set -e
set -x

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
echo "= preparing gpg"
GNUPGHOME="$(mktemp -d)"
export GNUPGHOME
# public key for musl
gpg --batch --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 836489290BB6B70F99FFDA0556BCDB593020450F

# download tarballs
echo "= downloading busybox"

busybox_dir="busybox-${busybox_version}"
if [ ! -e "${busybox_dir}" ]; then
  #https://github.com/mirror/busybox
  curl -LO "https://github.com/mirror/busybox/archive/refs/tags/${busybox_version}.tar.gz"

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
    curl -LO https://musl.libc.org/releases/musl-${musl_version}.tar.gz.asc
    gpg --batch --verify musl-${musl_version}.tar.gz.asc musl-${musl_version}.tar.gz
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

if [ ! -e ${busybox_dir}/busybox]; then
  export CFLAGS="-static"
  echo export CFLAGS="-static"

  echo "= building busybox"
  # .config incldues:
  # use exe applets, static, cross prefix = musl-, standalone shell
  # for menuconfig you may need to install `sudo dnf install make gcc autoconf ncurses-devel`
  cp ${script_dir}/.config ${busybox_dir}

  pushd ${busybox_dir}
  make
  popd # ${busybox_dir}
fi
popd # build

if [ ! -d built ]; then
  mkdir built
fi

cp build/${busybox_dir}/busybox built/ash