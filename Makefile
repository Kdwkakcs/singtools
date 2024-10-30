# Define default build tags that are enabled
DEFAULT_BUILD_TAGS := with_quic with_dhcp with_wireguard with_ech with_utls with_reality_server with_acme with_gvisor with_utls

# Convert the build tags to go build tag format
BUILD_TAGS := $(addprefix -tags ,$(DEFAULT_BUILD_TAGS))

# The binary name
BINARY := singtools

# Pre-build step to clone the repository and build libsubconverter
.PHONY: prebuild
prebuild:
	git clone https://github.com/Blakfs24/subconverter.git
	cd subconverter && sudo bash ./scripts/build.ubuntu.library.sh
	cp subconverter/libsubconverter.a ./get/
	rm -rf subconverter

# Standard build
build:
	go build -ldflags="-s -w" -tags "with_quic with_dhcp with_wireguard with_ech with_utls with_reality_server with_acme with_gvisor with_utls" -o $(BINARY) ./cmd/

# All builds
.PHONY: all
all: prebuild build all 