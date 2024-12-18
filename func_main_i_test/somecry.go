package main

import (
	"fmt"
	"notes/mycrypto"
)

func main() {
	somestr := mycrypto.Generate_salt()
	fmt.Println(somestr)
}
