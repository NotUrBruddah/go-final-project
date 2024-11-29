package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var validFormats = []string{
	`^y$`, // Формат "y"
	`^d\s[1-9]$|^d\s([1-3]\d{1,2})$|^d\s400$`, // Формат "d <число от 1 до 400>"
}

func IsValidFormat(input string) bool {
	for _, pattern := range validFormats {
		match, _ := regexp.MatchString(pattern, input)
		if match {
			return true
		}
	}
	return false
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	//преобразуем date к формату time.Time
	startDate, err := time.Parse("20060102", date)
	if err != nil {
		return "", err
	}

	if repeat == "" {
		return "", fmt.Errorf("пустое значение repeat")
	}

	//валидируем repeat переменную
	if !(IsValidFormat(repeat)) {
		return "", fmt.Errorf("некорректный формат repeat")
	}

	substrs := strings.Split(repeat, " ")
	switch substrs[0] {
	case "y":
		nextDate := startDate.AddDate(1, 0, 0)
		for nextDate.Before(now) || nextDate.Equal(now) {
			nextDate = nextDate.AddDate(1, 0, 0)
		}
		return nextDate.Format("20060102"), nil
	case "d":
		days, _ := strconv.Atoi(substrs[1])
		nextDate := startDate.AddDate(0, 0, days)
		for nextDate.Before(now) || nextDate.Equal(now) {
			nextDate = nextDate.AddDate(0, 0, days)
		}
		return nextDate.Format("20060102"), nil
	case "w":
	case "m":
	default:
		return "", fmt.Errorf("неподдерживаемый формат")
	}
	return "", fmt.Errorf("unexpected error")
}
