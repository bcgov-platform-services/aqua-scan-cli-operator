package utils

import (
	"math/rand"
	"regexp"
)

func makePassword(length int, generationString []rune) string {
	s := make([]rune, length)
	for i := range s {
		s[i] = generationString[rand.Intn(len(generationString))]
	}
	return string(s)
}

func GeneratePassword(length int, hasSymbols bool, hasNumbers bool, hasUppercase bool) string {
	const (
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		numbers   = "0123456789"
		symbols   = "!@#$"
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	)

	reLowercase, _ := regexp.Compile("[a-z]+")
	reUppercase, _ := regexp.Compile("[A-Z]+")
	reNumbers, _ := regexp.Compile("[0-9]+")
	reSymbols, _ := regexp.Compile("[!@#$]+")

	var generationString = lowercase

	if hasUppercase {
		generationString = generationString + uppercase
	}

	if hasSymbols {
		generationString = generationString + symbols
	}

	if hasNumbers {
		generationString = generationString + numbers
	}

	var password string

	genStringRunes := []rune(generationString)

	for {
		password = makePassword(length, genStringRunes)
		pBytes := []byte(password)

		if hasNumbers {

			numbersMatch := reNumbers.Match(pBytes)
			if !numbersMatch {
				continue
			}
		}

		if hasSymbols {
			symbolsMatch := reSymbols.Match(pBytes)
			if !symbolsMatch {
				continue
			}
		}

		if hasUppercase {
			uppercaseMatches := reUppercase.Match(pBytes)
			if !uppercaseMatches {
				continue
			}
		}

		lowercaseMatches := reLowercase.Match(pBytes)

		if !lowercaseMatches {
			continue
		}

		break
	}
	return password
}
