package service

import (
	"regexp"
	"strings"
)

type ProfanityService struct {
	badWords []string
	re       *regexp.Regexp
}

func NewProfanityService() *ProfanityService {
	// Common Indonesian and English profanity
	// Note: This is an initial list, can be expanded or loaded from a file/DB
	words := []string{
		"anjing", "anjrit", "anjeng", "ancuk", "asu", "bangsat", "bego", "tolol", "goblok",
		"kontal", "kontol", "memek", "jembut", "peler", "itil", "ngewe", "entot",
		"perek", "lonte", "jablay", "pelacur", "pantek", "kimak", "fuck", "shit",
		"bitch", "asshole", "pussy", "dick", "bastard", "idiot", "ngentot", "taik",
		"tai", "asu", "bajingan", "brengsek", "setan", "iblis", "monyet", "babi",
	}

	return &ProfanityService{
		badWords: words,
		re:       createProfanityRegex(words),
	}
}

func createProfanityRegex(words []string) *regexp.Regexp {
	var escapedWords []string
	for _, w := range words {
		escapedWords = append(escapedWords, regexp.QuoteMeta(w))
	}
	// Case-insensitive match for the words
	pattern := `(?i)\b(` + strings.Join(escapedWords, "|") + `)\b`
	return regexp.MustCompile(pattern)
}

func (s *ProfanityService) Clean(text string) string {
	return s.re.ReplaceAllStringFunc(text, func(match string) string {
		return strings.Repeat("*", len(match))
	})
}

func (s *ProfanityService) IsBad(text string) bool {
	return s.re.MatchString(text)
}
