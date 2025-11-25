package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pquerna/otp/totp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run generate_otp.go <secret>")
		os.Exit(1)
	}

	secret := os.Args[1]
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		fmt.Printf("Error generating code: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(code)
}
