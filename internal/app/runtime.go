package app

import (
	"fmt"
	"os"

	"github.com/0robustus1/spirited-env/internal/config"
	"github.com/0robustus1/spirited-env/internal/pathmap"
)

type Runtime struct {
	Mapper    pathmap.Mapper
	Paths     config.Paths
	Settings  config.Settings
	ConfigErr error
}

func NewRuntime() *Runtime {
	paths, err := config.ResolvePaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	settings, configErr := config.LoadSettings(paths.ConfigFile)

	mapper, err := pathmap.New(paths.EnvironsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return &Runtime{Mapper: mapper, Paths: paths, Settings: settings, ConfigErr: configErr}
}
