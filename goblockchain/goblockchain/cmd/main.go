package main

import (
	"awesomeProject/utils"
	"fmt"
)

func main() {
	fmt.Println(utils.IsFoundHost("127.0.0.1", 5001))
}
