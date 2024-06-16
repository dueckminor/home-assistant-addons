package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	done := make(chan bool)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <- sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	fmt.Println("gateway started...")

	<- done

	fmt.Println("gateway stopped...")
}
