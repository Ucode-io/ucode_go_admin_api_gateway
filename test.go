package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {

	aaa, err := os.ReadFile("./test.txt")
	if err != nil {
		panic(err)
	}

	var dddd map[string]any

	err = json.Unmarshal(aaa, &dddd)
	if err != nil {
		panic(err)
	}

	by, err := json.Marshal(dddd)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(by))
}
