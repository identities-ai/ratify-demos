# Mandate Demo — Pi Physical Verifier
#
# Run on the Raspberry Pi. Requires the Ratify C SDK to be built first.
#
# First-time setup:
#   cd /path/to/ratify-protocol/sdks/c && cargo build --release
#
# Then from this directory:
#   SDK_DIR=/path/to/ratify-protocol/sdks/c make
#
# Or set SDK_DIR in your shell:
#   export SDK_DIR=~/projects/IdentitiesAI/ratify-protocol/sdks/c

SDK_DIR ?= $(HOME)/projects/IdentitiesAI/ratify-protocol/sdks/c
LIB     := $(SDK_DIR)/target/release/libratify_c.a
INCLUDE := $(SDK_DIR)/include

CC      := gcc
CFLAGS  := -Wall -Wextra -std=c99 -O2 -I$(INCLUDE)
LDFLAGS := $(LIB) -lpthread -ldl -lm

.PHONY: all clean check-sdk

all: check-sdk verifier

check-sdk:
	@if [ ! -f "$(LIB)" ]; then \
		echo ""; \
		echo "ERROR: C SDK not built. Run:"; \
		echo "  cd $(SDK_DIR) && cargo build --release"; \
		echo ""; \
		exit 1; \
	fi

verifier: verifier.c $(LIB)
	$(CC) $(CFLAGS) -o $@ $< $(LDFLAGS)
	@echo "Built: ./verifier"
	@echo "Run:   BUNDLE_FILE=bundle.json ./verifier"
	@echo "Test:  BUNDLE_FILE=bundle.json NO_GPIO=1 ./verifier"

clean:
	rm -f verifier
