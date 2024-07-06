package main

import (
	"flag"

	"github.com/dueckminor/home-assistant-addons/go/auth"
)

var dataDir string

func init() {
	flag.StringVar(&dataDir, "data-dir", "/data", "the data dir")
	flag.Parse()
}

func main() {
	u, err := auth.NewUsers(dataDir)
	if err != nil {
		panic(err)
	}

	u.AddUser(flag.Arg(0), flag.Arg(1))
}
