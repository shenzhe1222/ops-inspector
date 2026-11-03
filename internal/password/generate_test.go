package password

import (
	"strconv"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	for _, length := range []int{minPasswordLength, 16, maxPasswordLength} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			generated, err := Generate(length)
			if err != nil {
				t.Fatalf("Generate(%d) error = %v", length, err)
			}
			if len(generated) != length {
				t.Fatalf("Generate(%d) length = %d", length, len(generated))
			}
			for _, character := range generated {
				if !strings.ContainsRune(uppercase+lowercase+digits, character) {
					t.Errorf("Generate(%d) contains invalid character %q", length, character)
				}
			}
			if !strings.ContainsAny(generated, uppercase) {
				t.Errorf("Generate(%d) contains no uppercase letter", length)
			}
			if !strings.ContainsAny(generated, lowercase) {
				t.Errorf("Generate(%d) contains no lowercase letter", length)
			}
			if !strings.ContainsAny(generated, digits) {
				t.Errorf("Generate(%d) contains no digit", length)
			}
		})
	}
}

func TestGenerateRejectsInvalidLength(t *testing.T) {
	for _, length := range []int{0, 8, 64} {
		if _, err := Generate(length); err == nil {
			t.Errorf("Generate(%d) error = nil, want error", length)
		}
	}
}
