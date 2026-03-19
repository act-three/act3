#!/bin/bash
set -euo pipefail

f=$(mktemp /tmp/act3-mcp.XXXXXX)
go build -C cmd/act3-mcp -o $f .
exec $f
