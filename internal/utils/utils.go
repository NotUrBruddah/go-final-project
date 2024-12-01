package utils

import (
	"regexp"
)

func IsValidFormat(input string, patterns []string) bool {
	for _, pattern := range patterns {
		match, _ := regexp.MatchString(pattern, input)
		if match {
			return true
		}
	}
	return false
}
