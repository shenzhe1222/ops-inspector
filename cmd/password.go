package cmd

import (
	"fmt"
	"strconv"

	"ops-inspector/internal/password"
)

func runPassword(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ops-inspector -P <length>")
	}

	length, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid password length %q: must be an integer between 8 and 64", args[0])
	}

	generated, err := password.Generate(length)
	if err != nil {
		return err
	}

	fmt.Println(generated)
	return nil
}
