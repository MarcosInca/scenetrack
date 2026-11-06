#!/bin/sh
set -a; . ./.env; set +a
exec go run ./cmd/api
