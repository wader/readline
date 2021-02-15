package main

import (
	"fmt"

	"github.com/goinsane/readline/v2"
)

func main() {
	t, err := readline.NewTerminal(readline.Config{})
	defer t.Close()
	if err != nil {
		panic(err)
	}
	for {
		p, err := t.ReadSlice()
		if p != nil {
			fmt.Println(string(p))
		}
		if err != nil {
			fmt.Println(err)
			break
		}
	}
}
