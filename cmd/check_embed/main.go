package main

import (
	"crypto/md5"
	"fmt"

	"github.com/decker502/pvz/embedded"
)

func main() {
	data, err := embedded.ReadFile("assets/sounds/evillaugh.ogg")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	hash := md5.Sum(data)
	fmt.Printf("Embedded evillaugh.ogg MD5: %x\n", hash)
	fmt.Printf("File size: %d bytes\n", len(data))
}
