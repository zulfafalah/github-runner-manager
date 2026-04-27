package main

import (
	"log"

	"github-runner-manager/ui"
)

func main() {
	app := ui.NewApp()
	app.Initialize()

	log.Println("Starting GitHub Runner Manager...")
	app.Run()
}
