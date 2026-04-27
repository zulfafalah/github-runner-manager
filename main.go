package main

import (
	"log"

	"github-runner-manager/ui"
)

func main() {
	// Buat instance aplikasi
	app := ui.NewApp()
	
	// Inisialisasi UI dan komponen
	app.Initialize()
	
	// Jalankan aplikasi
	log.Println("Starting GitHub Runner Manager...")
	app.Run()
}
