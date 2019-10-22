#!/bin/bash

OS=`uname -o`
ARCH=`uname -m`

case $OS in
    GNU/Linux) echo $ARCH-linux-gnu; exit 0 ;;
	Cygwin)	   echo $ARCH-win-mingw; exit 0 ;;
esac
