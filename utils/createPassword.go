package utils

import (
	"math/rand"
	"time"
)

// https://alan-g-bardales.medium.com/password-generator-with-go-golang-c31190121008
func CreatePassword(length int) string {
	const voc string = "abcdfghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const numbers string = "0123456789"
	const symbols string = "!@#$%&*+_-="

	chars := voc

	chars = chars + numbers

	chars = chars + symbols

	return generatePassword(length, chars)
}

func generatePassword(length int, chars string) string {
	password := ""
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	for i := 0; i < length; i++ {
		password += string([]rune(chars)[r.Intn(len(chars))])
	}
	return password
}
