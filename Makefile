
rootdir = $(PWD)
sys = $(shell ./scripts/sys_name.sh)

export GOPATH = $(shell printenv GOPATH):$(rootdir)

$(rootdir)/output/.intermediate:
	mkdir -p $@

$(rootdir)/output/.intermediate/rtmpdump: $(rootdir)/output/.intermediate
	cd $(rootdir)/output/.intermediate; git clone git://git.ffmpeg.org/rtmpdump

$(rootdir)/output/.intermediate/rtmpdump/.patehed: $(rootdir)/output/.intermediate/rtmpdump
	cd $(rootdir)/output/.intermediate; patch -p0 < $(rootdir)/patch/librtmp.patch

$(rootdir)/output/$(sys)/lib/librtmp.a: $(rootdir)/output/.intermediate/rtmpdump $(rootdir)/output/.intermediate/rtmpdump/.patehed
	cd $(rootdir)/output/.intermediate/rtmpdump; make prefix=$(rootdir)/output/$(sys) CRYPTO= clean
	cd $(rootdir)/output/.intermediate/rtmpdump; make prefix=$(rootdir)/output/$(sys) CRYPTO= all
	cd $(rootdir)/output/.intermediate/rtmpdump; make prefix=$(rootdir)/output/$(sys) CRYPTO= install

all: $(rootdir)/output/$(sys)/lib/librtmp.a
	cd src; CGO_ENABLED=1 	\
		CGO_LDFLAGS=-L$(rootdir)/output/$(sys)/lib		\
		CGO_CFLAGS=-I$(rootdir)/output/$(sys)/include	\
		go build -o $(rootdir)/output/$(sys)/bin/go-rtmp-server
