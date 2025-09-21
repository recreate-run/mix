package main

import (
	"mix/cmd"
	"mix/internal/logging"
)

func main() {
	defer logging.RecoverPanic("main", func() {
		logging.Error("Application terminated due to unhandled panic")
	})

	cmd.Execute()
}
