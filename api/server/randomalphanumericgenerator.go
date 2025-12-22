package server

import "math/rand"

const alphanumericChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type RandomAlphanumericGenerator interface {
	Generate(length int) string
}

type randomAlphanumericGenerator struct{}

func NewRandomAlphanumericGenerator() RandomAlphanumericGenerator {
	return &randomAlphanumericGenerator{}
}

func (r *randomAlphanumericGenerator) Generate(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = alphanumericChars[rand.Intn(len(alphanumericChars))]
	}
	return string(b)
}
