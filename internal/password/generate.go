package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	minPasswordLength = 9
	maxPasswordLength = 63
	uppercase         = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercase         = "abcdefghijklmnopqrstuvwxyz"
	digits            = "0123456789"
)

func Generate(length int) (string, error) {
	if length < minPasswordLength || length > maxPasswordLength {
		return "", fmt.Errorf("password length must be greater than 8 and less than 64")
	}

	password := make([]byte, 0, length)
	for _, characters := range []string{uppercase, lowercase, digits} {
		index, err := randomIndex(len(characters))
		if err != nil {
			return "", err
		}
		password = append(password, characters[index])
	}

	characters := uppercase + lowercase + digits
	for len(password) < length {
		index, err := randomIndex(len(characters))
		if err != nil {
			return "", err
		}
		password = append(password, characters[index])
	}

	for i := len(password) - 1; i > 0; i-- {
		index, err := randomIndex(i + 1)
		if err != nil {
			return "", err
		}
		password[i], password[index] = password[index], password[i]
	}

	return string(password), nil
}

func randomIndex(max int) (int, error) {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, fmt.Errorf("generate secure random number: %w", err)
	}
	return int(index.Int64()), nil
}
