// main.go

package main

import (
	"fmt"
	"os"
)

func main() {
	a := App{}
	fmt.Println("Project begin")
	a.Initialize(
		os.Getenv("APP_DB_USERNAME"),
		os.Getenv("APP_DB_PASSWORD"),
		os.Getenv("APP_DB_NAME"))

	a.Run(":8010")
}
