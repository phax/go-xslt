package main

// appVersion is the version of this build: "dev" for a plain `go build` or
// `wails3 dev`, the release version (e.g. "0.1.6") for a build made through
// the Makefile, which writes version_stamp.go (gitignored) before building.
var appVersion = "dev"
