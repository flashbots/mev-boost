# mev-boost

A middleware server written in Go, that sits between an ethereum PoS consensus client and an execution client. It allows consensus clients to recieve bundles from proposers as well as fallback to execution clients. See https://hackmd.io/I-F1RiphRK-HhT65TX1cpQ for the current spec.

## Build

```
make build
```

and then run it with:

```
./mev-boost
```

## Test

```
make test
```

## Lint

We use `revive` as a linter. You need to install it with `go install github.com/mgechev/revive@latest`

```
make lint
```
