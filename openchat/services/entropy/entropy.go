package entropy

import (
	"strings"

	"github.com/ipfn/go-mnemonic/mnemonic"
)

func GeneratePair() (success bool, entropy []byte, words []string, err error) {
	// Generate 256 bits of entropy
	entropy, err = mnemonic.NewEntropy(256)
	if err != nil {
		return false, entropy, words, err
	}
	words, err = GenerateWords(entropy)
	if err != nil {
		return false, entropy, words, err
	}
	return true, entropy, words, nil
}

func GenerateWords(entropy []byte) (words []string, err error) {
	// Convert entropy to a 24-word mnemonic
	phrase, err := mnemonic.New(entropy)
	if err != nil {
		return words, err
	}

	words = strings.Fields(phrase)
	return words, nil
}

func VerifyWords(words []string)
