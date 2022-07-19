#!/bin/bash

set -x
set -e

cp velux-nibe synology/bin
cd synology
./INFO.sh > INFO
tar cvfz package.tgz bin
tar -c -v --exclude INFO.sh --exclude repo --exclude velux-nibe*.spk \
  -f velux-nibe-"${SPK_ARCH:-x86_64}"-"${SPK_PACKAGE_SUFFIX:-latest}".spk \
  *
