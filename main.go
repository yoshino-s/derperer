package main

import (
	"github.com/yoshino-s/derperer/cmd"
)

//go:generate go tool swag init --output ./internal/handler/http/docs

func main() {
	cmd.Execute()
}
