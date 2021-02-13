package main

import (
	"fmt"
	"os"

	"github.com/goinsane/readline/v2"
)

func main() {
	//stIn, _ := readline.MakeRaw(os.Stdin.Fd())
	//defer readline.SetState(os.Stdin.Fd(), stIn)

	//stOut, _ := readline.MakeRaw(os.Stdin.Fd())
	//defer readline.SetState(os.Stdout.Fd(), stOut)

	a, _ := readline.ReadPassword(os.Stdin.Fd())
	fmt.Println(string(a))

	//var str string
	//fmt.Scanln(&str)
	//fmt.Println(str)
	//time.Sleep(5 * time.Second)
}
