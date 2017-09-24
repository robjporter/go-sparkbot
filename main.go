package main

import (
	"./app"
)

func main() {
	a := app.New()
	a.LoadConfig("conf/config.yaml")
	a.Run()
}
