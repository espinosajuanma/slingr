#!/bin/bash
go build -o dist/slingr-go ./cmd/slingr
install -m 0755 dist/slingr-go "$HOME/.local/bin/slingr-go"
