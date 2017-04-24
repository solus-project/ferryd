PROJECT_ROOT := src/
VERSION = 0.0.1

.DEFAULT_GOAL := all

# CLI app
ferry:
	GOPATH=$(CUR_DIR) go install -v cli && mv bin/cli bin/ferry

ferryd:
	GOPATH=$(CUR_DIR) go install -v daemon && mv bin/daemon bin/ferryd

BINS = \
	ferry \
	ferryd

GO_TESTS = \
	libeopkg.test


include Makefile.gobuild

_PKGS = \
	cli \
	cli/cmd \
	ferry \
	daemon \
	daemon/server \
	libeopkg


# We want to add compliance for all built binaries
_CHECK_COMPLIANCE = $(addsuffix .compliant,$(_PKGS))

# Ensure our own code is compliant..
compliant: $(_CHECK_COMPLIANCE)
install: $(BINS)
	test -d $(DESTDIR)/usr/bin || install -D -d -m 00755 $(DESTDIR)/usr/bin; \
	install -m 00755 bin/* $(DESTDIR)/usr/bin/.;

ensure_modules:
	@ ( \
		git submodule init; \
		git submodule update; \
	);

# See: https://github.com/meitar/git-archive-all.sh/blob/master/git-archive-all.sh
release: ensure_modules
	git-archive-all.sh --format tar.gz --prefix ferryd-$(VERSION)/ --verbose -t HEAD ferryd-$(VERSION).tar.gz

all: $(BINS)
