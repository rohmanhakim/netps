package main

import (
	"fmt"
	"netps/internal/parser"
)

func main() {
	res, err := parser.ScanListeningPortsProcfs()

	if err != nil {
		panic(err)
	}

	fmt.Printf("result: %v", res)

}
