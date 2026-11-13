package main

import (
	"github.com/jinghaihan/feth-save-editor/internal/app"
	"log"
	"os"
)

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if err := app.Run(path); err != nil {
		log.Fatal(err)
	}
}
