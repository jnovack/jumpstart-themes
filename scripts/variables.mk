APPLICATION  = decklist-print
VERSION      = $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
REVISION     = $(shell git rev-parse --short HEAD 2>/dev/null || echo local)
BUILD_DATE   = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
