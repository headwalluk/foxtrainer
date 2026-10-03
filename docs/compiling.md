# Compiling from source

Most people don't need this: [Getting started](getting-started.md) installs a pre-built Linux
binary in one command. Build from source if you want to read the code you run, need a platform
without a pre-built binary (such as macOS), or want an unreleased version.

foxtrainer is a single Go program with no C dependencies, so the result is one static binary.

## Install Go

foxtrainer needs **Go 1.26 or later**. Follow the official instructions at
[go.dev/doc/install](https://go.dev/doc/install), which cover Linux, macOS and Windows, then
check it works:

```console
$ go version
go version go1.26.2 linux/amd64
```

A distribution's Go package also works if it is Go 1.21 or later, such as `golang-go` on Debian
13 (Go 1.24). When a project needs a newer Go, the `go` command downloads that toolchain
automatically the first time. This doesn't happen if `GOTOOLCHAIN=local` is set; check with
`go env GOTOOLCHAIN`, which should say `auto`.

## Option 1: `go install`

This fetches, builds and installs a tagged release in one step:

```console
$ go install github.com/headwalluk/foxtrainer/cmd/foxtrainer@latest
```

Use `@v1.0.0` instead of `@latest` for a particular release. The binary goes into Go's own bin
folder, `~/go/bin` unless you've changed `GOBIN` or `GOPATH` (`go env GOPATH` shows where).
Add that folder to your `PATH`, or move the binary somewhere that is:

```console
$ mv ~/go/bin/foxtrainer ~/.local/bin/
```

## Option 2: Clone and build

To build a checkout, a branch or your own changes:

```console
$ git clone https://github.com/headwalluk/foxtrainer.git
$ cd foxtrainer
$ git checkout v1.0.0        # optional: a release tag; omit to build the latest code
$ make build                 # writes bin/foxtrainer
$ install -m 0755 bin/foxtrainer ~/.local/bin/
```

`make build` needs only Go and `make`. It builds with the same flags as a release and stamps the
version from `git describe`. Without `make`, this works too:

```console
$ go build -trimpath -o bin/foxtrainer ./cmd/foxtrainer
```

## Check the build

```console
$ foxtrainer version
foxtrainer 1.0.0
```

A build from a commit after a tag shows something like `1.0.0-3-gabc1234`, and `-dirty` is added
when the checkout has uncommitted changes.

## Building for another machine

Go cross-compiles without extra tools. For example, a binary for a 64-bit ARM Linux machine:

```console
$ CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o foxtrainer-arm64 ./cmd/foxtrainer
```

## macOS

foxtrainer builds and runs on macOS, but is untested there in 1.0. It looks for profiles in
`~/Library/Application Support/Firefox`, but it doesn't yet look for Firefox installs in
`/Applications`, so it may show no instances. Spellcheck dictionaries are set up on Linux only.

## Next

- [Getting started](getting-started.md#first-run): first run and everyday use.
- [Contributing](contributing.md): linting, the test suites and code conventions, if you want to
  change foxtrainer.
