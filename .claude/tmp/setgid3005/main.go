package main

import (
	"fmt"
	"os"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: setgid3005 <binary> [args...]")
		os.Exit(1)
	}
	if err := syscall.Setgid(3005); err != nil {
		fmt.Fprintf(os.Stderr, "setgid(3005): %v\n", err)
		os.Exit(1)
	}
	if err := syscall.Exec(os.Args[1], os.Args[1:], os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		os.Exit(1)
	}
}
