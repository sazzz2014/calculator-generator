//go:build !cgo || !linux

package native

import "errors"

func New() (Calculator, error) {
	return nil, errors.New("native calculator requires Linux with cgo and the Rust static library; use docker compose up --build")
}
