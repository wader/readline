package main

import (
	"fmt"

	"github.com/goinsane/readline/v2"
)

func main() {
	t, err := readline.NewTerminal(readline.Config{
		Prompt: "> ",
	})
	defer t.Close()
	if err != nil {
		panic(err)
	}
	for {
		p, err := t.ReadBytes()
		if p != nil {
			fmt.Println(string(p))
		}
		if err != nil {
			fmt.Println(err)
			break
		}
		//fmt.Println(len(p))
	}
}
