//go:build !(js && wasm)

// This package is the browser WASM playground; it is meant to be built with
// GOOS=js GOARCH=wasm (see `make playground`). On every other platform this stub
// keeps the package buildable so `go build ./...` and CI stay green.
package main

import "fmt"

func main() {
	fmt.Println("build the playground with: GOOS=js GOARCH=wasm go build -o playground/keyway.wasm ./playground")
}
