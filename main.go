package main

import (
	_ "time/tzdata" // embed IANA timezone database for containers without tzdata

	"github.com/mnh-shop/7alosclaw/cmd"
)

func main() {
	cmd.Execute()
}
