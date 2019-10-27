
rootdir = $(shell pwd)
sys = $(shell ./scripts/sys_name.sh)

export GOPATH = $(shell printenv GOPATH):$(rootdir)

libs = $(rootdir)/output/$(sys)/lib/librtmp.a
dirs = $(rootdir)/output/.intermediate/rtmpdump

$(rootdir)/output/.intermediate:
	mkdir -p $@

$(rootdir)/output/.intermediate/rtmpdump: $(rootdir)/output/.intermediate
	cd $(rootdir)/output/.intermediate; git clone git://git.ffmpeg.org/rtmpdump

$(rootdir)/output/.intermediate/.rtmp.patched:
	cd $(rootdir)/output/.intermediate; patch -p0 < $(rootdir)/patch/librtmp.patch
	touch $@

$(rootdir)/output/$(sys)/lib/librtmp.a: $(rootdir)/output/.intermediate/.rtmp.patched
	cd $(rootdir)/output/.intermediate/rtmpdump; make prefix=$(rootdir)/output/$(sys) CRYPTO= clean
	cd $(rootdir)/output/.intermediate/rtmpdump; make prefix=$(rootdir)/output/$(sys) CRYPTO= XCFLAGS=-g OPT=-O0 all
	cd $(rootdir)/output/.intermediate/rtmpdump; make prefix=$(rootdir)/output/$(sys) CRYPTO= install

prepare: $(dirs)

deps: prepare $(libs)

export CGO_CFLAGS=-I$(rootdir)/output/$(sys)/include
export CGO_LDFLAGS=-L$(rootdir)/output/$(sys)/lib -lrtmp

all: $(libs)
	cd src; CGO_ENABLED=1 	\
		go build -x -o $(rootdir)/output/$(sys)/bin/go-rtmp-server

.PHONY: all
