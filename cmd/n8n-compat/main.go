package main

import "log"

func main() {
	if err := newRootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
