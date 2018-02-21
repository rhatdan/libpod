GO ?= go
EPOCH_TEST_COMMIT ?= 9e3a5da6
HEAD ?= HEAD
PROJECT := github.com/projectatomic/libpod
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
GIT_BRANCH_CLEAN := $(shell echo $(GIT_BRANCH) | sed -e "s/[^[:alnum:]]/-/g")
LIBPOD_IMAGE := libpod_dev$(if $(GIT_BRANCH_CLEAN),:$(GIT_BRANCH_CLEAN))
LIBPOD_INSTANCE := libpod_dev
PREFIX ?= ${DESTDIR}/usr/local
BINDIR ?= ${PREFIX}/bin
LIBEXECDIR ?= ${PREFIX}/libexec
MANDIR ?= ${PREFIX}/share/man
ETCDIR ?= ${DESTDIR}/etc
ETCDIR_LIBPOD ?= ${ETCDIR}/crio
BUILDTAGS ?= seccomp $(shell hack/btrfs_tag.sh) $(shell hack/libdm_tag.sh) $(shell hack/btrfs_installed_tag.sh) $(shell hack/ostree_tag.sh) $(shell hack/selinux_tag.sh)

BASHINSTALLDIR=${PREFIX}/share/bash-completion/completions
OCIUMOUNTINSTALLDIR=$(PREFIX)/share/oci-umount/oci-umount.d

SELINUXOPT ?= $(shell test -x /usr/sbin/selinuxenabled && selinuxenabled && echo -Z)
PACKAGES ?= $(shell go list -tags "${BUILDTAGS}" ./... | grep -v github.com/projectatomic/libpod/vendor | grep -v e2e)

COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
GIT_COMMIT := $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")
BUILD_INFO := $(shell date +%s)

# If GOPATH not specified, use one in the local directory
ifeq ($(GOPATH),)
export GOPATH := $(CURDIR)/_output
unexport GOBIN
endif
GOPKGDIR := $(GOPATH)/src/$(PROJECT)
GOPKGBASEDIR := $(shell dirname "$(GOPKGDIR)")

# Update VPATH so make finds .gopathok
VPATH := $(VPATH):$(GOPATH)
SHRINKFLAGS := -s -w
BASE_LDFLAGS := ${SHRINKFLAGS} -X main.gitCommit=${GIT_COMMIT} -X main.buildInfo=${BUILD_INFO}
LDFLAGS := -ldflags '${BASE_LDFLAGS}'
LDFLAGS_PODMAN := -ldflags '${BASE_LDFLAGS}'

BOX="fedora_atomic"

all: binaries docs

default: help

help:
	@echo "Usage: make <target>"
	@echo
	@echo " * 'install' - Install binaries to system locations"
	@echo " * 'binaries' - Build podmon"
	@echo " * 'integration' - Execute integration tests"
	@echo " * 'clean' - Clean artifacts"
	@echo " * 'lint' - Execute the source code linter"
	@echo " * 'gofmt' - Verify the source code gofmt"

.gopathok:
ifeq ("$(wildcard $(GOPKGDIR))","")
	mkdir -p "$(GOPKGBASEDIR)"
	ln -s "$(CURDIR)" "$(GOPKGBASEDIR)"
endif
	touch "$(GOPATH)/.gopathok"

lint: .gopathok
	@echo "checking lint"
	@./.tool/lint

gofmt:
	@./hack/verify-gofmt.sh

fix_gofmt:
	@./hack/verify-gofmt.sh -f

test/bin2img/bin2img: .gopathok $(wildcard test/bin2img/*.go)
	$(GO) build $(LDFLAGS) -tags "$(BUILDTAGS) containers_image_ostree_stub" -o $@ $(PROJECT)/test/bin2img

test/copyimg/copyimg: .gopathok $(wildcard test/copyimg/*.go)
	$(GO) build $(LDFLAGS) -tags "$(BUILDTAGS) containers_image_ostree_stub" -o $@ $(PROJECT)/test/copyimg

test/checkseccomp/checkseccomp: .gopathok $(wildcard test/checkseccomp/*.go)
	$(GO) build $(LDFLAGS) -tags "$(BUILDTAGS) containers_image_ostree_stub" -o $@ $(PROJECT)/test/checkseccomp

podman: .gopathok $(shell hack/find-godeps.sh $(GOPKGDIR) cmd/podman $(PROJECT))
	$(GO) build -i $(LDFLAGS_PODMAN) -tags "$(BUILDTAGS)" -o bin/$@ $(PROJECT)/cmd/podman

clean:
ifneq ($(GOPATH),)
	rm -f "$(GOPATH)/.gopathok"
endif
	rm -rf _output
	rm -f docs/*.1
	rm -fr test/testdata/redis-image
	find . -name \*~ -delete
	find . -name \#\* -delete
	rm -f bin/podman
	rm -f test/bin2img/bin2img
	rm -f test/copyimg/copyimg
	rm -f test/checkseccomp/checkseccomp

libpodimage:
	docker build -t ${LIBPOD_IMAGE} .

dbuild: libpodimage
	docker run --name=${LIBPOD_INSTANCE} --privileged ${LIBPOD_IMAGE} -v ${PWD}:/go/src/${PROJECT} --rm make binaries

integration: libpodimage
	docker run -e STORAGE_OPTIONS="--storage-driver=vfs" -e TESTFLAGS -e TRAVIS -t --privileged --rm -v ${CURDIR}:/go/src/${PROJECT} ${LIBPOD_IMAGE} make localintegration

integration.fedora:
	DIST=Fedora sh .papr_prepare.sh

integration.centos:
	DIST=CentOS sh .papr_prepare.sh

testunit:
	$(GO) test -tags "$(BUILDTAGS)" -cover $(PACKAGES)

ginkgo:
	ginkgo -v test/e2e/

localintegration: test-binaries
	ginkgo -v -cover -flakeAttempts 3 -progress -trace -noColor test/e2e/.

vagrant-check:
	BOX=$(BOX) sh ./vagrant.sh

binaries: podman

test-binaries: test/bin2img/bin2img test/copyimg/copyimg test/checkseccomp/checkseccomp

MANPAGES_MD := $(wildcard docs/*.md)
MANPAGES    := $(MANPAGES_MD:%.md=%)

docs/%.1: docs/%.1.md .gopathok
	(go-md2man -in $< -out $@.tmp && touch $@.tmp && mv $@.tmp $@) || ($(GOPATH)/bin/go-md2man -in $< -out $@.tmp && touch $@.tmp && mv $@.tmp $@)

docs: $(MANPAGES)

docker-docs: docs
	(cd docs; ./dckrman.sh *.1)

install: .gopathok install.bin install.man install.cni

install.bin:
	install ${SELINUXOPT} -D -m 755 bin/podman $(BINDIR)/podman

install.man: docs
	install ${SELINUXOPT} -d -m 755 $(MANDIR)/man1
	install ${SELINUXOPT} -m 644 $(filter %.1,$(MANPAGES)) -t $(MANDIR)/man1

install.config:
	install ${SELINUXOPT} -D -m 644 seccomp.json $(ETCDIR_LIBPOD)/seccomp.json
	install ${SELINUXOPT} -D -m 644 crio-umount.conf $(OCIUMOUNTINSTALLDIR)/crio-umount.conf

install.completions:
	install ${SELINUXOPT} -d -m 755 ${BASHINSTALLDIR}
	install ${SELINUXOPT} -m 644 -D completions/bash/podman ${BASHINSTALLDIR}

install.cni:
	install ${SELINUXOPT} -D -m 644 cni/87-podman-bridge.conflist ${ETCDIR}/cni/net.d/87-podman-bridge.conflist

install.docker: docker-docs
	install ${SELINUXOPT} -D -m 755 docker $(BINDIR)/docker
	install ${SELINUXOPT} -d -m 755 $(MANDIR)/man1
	install ${SELINUXOPT} -m 644 docs/docker*.1 -t $(MANDIR)/man1

uninstall:
	for i in $(filter %.1,$(MANPAGES)); do \
		rm -f $(MANDIR)/man1/$$(basename $${i}); \
	done

.PHONY: .gitvalidation
.gitvalidation: .gopathok
	GIT_CHECK_EXCLUDE="./vendor" $(GOPATH)/bin/git-validation -v -run DCO,short-subject,dangling-whitespace -range $(EPOCH_TEST_COMMIT)..$(HEAD)

.PHONY: install.tools

install.tools: .install.gitvalidation .install.gometalinter .install.md2man

.install.gitvalidation: .gopathok
	if [ ! -x "$(GOPATH)/bin/git-validation" ]; then \
		go get -u github.com/vbatts/git-validation; \
	fi

.install.gometalinter: .gopathok
	if [ ! -x "$(GOPATH)/bin/gometalinter" ]; then \
		go get -u github.com/alecthomas/gometalinter; \
		cd $(GOPATH)/src/github.com/alecthomas/gometalinter; \
		git checkout 23261fa046586808612c61da7a81d75a658e0814; \
		go install github.com/alecthomas/gometalinter; \
		$(GOPATH)/bin/gometalinter --install; \
	fi

.install.md2man: .gopathok
	if [ ! -x "$(GOPATH)/bin/go-md2man" ]; then \
		   go get -u github.com/cpuguy83/go-md2man; \
	fi

.install.ostree: .gopathok
	if ! pkg-config ostree-1 2> /dev/null ; then \
		git clone https://github.com/ostreedev/ostree $(GOPATH)/src/github.com/ostreedev/ostree ; \
		cd $(GOPATH)/src/github.com/ostreedev/ostree ; \
		./autogen.sh --prefix=/usr/local; \
		make all install; \
	fi

.PHONY: \
	binaries \
	clean \
	default \
	docs \
	gofmt \
	help \
	install \
	lint \
	pause \
	uninstall
