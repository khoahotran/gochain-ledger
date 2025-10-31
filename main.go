package main

import (
	"os"

	"github.com/khoahotran/gochain-ledger/cmd"
)

func main() {

	defer func() {

	}()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
