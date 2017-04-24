PROJECT_ROOT := src/
VERSION = 0.0.1

.DEFAULT_GOAL := all

# The resulting binaries map to the subproject names
BINARIES = \
	ferry \
	ferryd

GO_TESTS = \
	libeopkg.test


include Makefile.gobuild

_PKGS = \
	ferry \
	ferry/cmd \
	ferryc \
	ferryd \
	ferryd/server \
	libeopkg


# We want to add compliance for all built binaries
_CHECK_COMPLIANCE = $(addsuffix .compliant,$(_PKGS))

# Build all binaries as static binary
BINS = $(addsuffix .build,$(BINARIES))

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
