package main

import "fmt"

func handleError(err error, msg string) {
	if err != nil {
		fmt.Printf(msg, err)
	}
}
