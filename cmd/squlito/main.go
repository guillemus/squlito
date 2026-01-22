package main

import (
	"fmt"
	"os"

	_ "modernc.org/sqlite"

	"squlito/internal/app"
	"squlito/internal/db"
)

func main() {
	dbPath := db.ParseDatabasePathFromArgs(os.Args[1:])
	err := app.Run(dbPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
