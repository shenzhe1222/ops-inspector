package cmd

import (
	"fmt"
	"strconv"

	"ops-inspector/internal/password"
)

func runPassword(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ops-inspector -P generate <length>")
	}
	if len(args) != 2 || args[0] != "generate" {
		return fmt.Errorf("usage: ops-inspector -P generate <length>")
	}

	length, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid password length %q: must be an integer between 8 and 64", args[1])
	}

	generated, err := password.Generate(length)
	if err != nil {
		return err
	}

	fmt.Println(generated)
	return nil
}
