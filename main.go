package main

import (
	_ "time/tzdata" // embed IANA timezone database for containers without tzdata

	"github.com/nextlevelbuilder/zalosclaw/cmd"
)

func main() {
	cmd.Execute()
}
