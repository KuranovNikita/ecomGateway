package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")

	cp := os.Getenv("CONFIG_PATH")

	fmt.Println(cp)
}
