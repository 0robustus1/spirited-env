package main

import (
	"fmt"
	"os"

	"github.com/0robustus1/spirited-env/internal/app"
	"github.com/alecthomas/kong"
)

type CLI struct {
	Path       app.PathCmd       `cmd:"" help:"Print mapped env file path."`
	Edit       app.EditCmd       `cmd:"" help:"Open mapped env file in $EDITOR."`
	Load       app.LoadCmd       `cmd:"" help:"Emit shell commands for loading env."`
	Refresh    app.RefreshCmd    `cmd:"" help:"Refresh env based on current location and state."`
	NoEnvExec  app.NoEnvExecCmd  `cmd:"" help:"Execute command with spirited-env managed vars unloaded."`
	Status     app.StatusCmd     `cmd:"" help:"Show discovered env file and key info."`
	Move       app.MoveCmd       `cmd:"" help:"Move mapped env file to a new directory mapping."`
	Import     app.ImportCmd     `cmd:"" help:"Import env assignments from existing file into spirited-env mapping."`
	Migrate    app.MigrateCmd    `cmd:"" help:"Import env assignments and move source file to centralized backup."`
	Config     app.ConfigCmd     `cmd:"" help:"Show effective configuration."`
	State      app.StateCmd      `cmd:"" help:"Inspect or reset internal shell state."`
	Init       app.InitCmd       `cmd:"" help:"Print shell integration snippet."`
	Completion app.CompletionCmd `cmd:"" help:"Print or install shell completion definitions."`
	Doctor     app.DoctorCmd     `cmd:"" help:"Run health checks for spirited-env setup."`
	Version    app.VersionCmd    `cmd:"" help:"Print version information."`
}

func main() {
	ctx := kong.Parse(&CLI{}, kong.Name("spirited-env"), kong.Description("Directory-based environment loader"))
	err := ctx.Run(app.NewRuntime())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
