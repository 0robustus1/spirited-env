package app

import (
	"fmt"
	"os"

	"github.com/0robustus1/spirited-env/internal/pathmap"
)

const (
	dirMode  = 0o700
	fileMode = 0o600
)

type Runtime struct {
	Mapper pathmap.Mapper
}

func NewRuntime() *Runtime {
	mapper, err := pathmap.New("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return &Runtime{Mapper: mapper}
}
