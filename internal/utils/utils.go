package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"
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

// Функция для преобразования слайса строк в слайс целых чисел
func StringToInt(slice []string) ([]int, error) {
	intSlice := make([]int, len(slice))
	var err error
	for i, s := range slice {
		intSlice[i], err = strconv.Atoi(s)
		if err != nil {
			return nil, err
		}
	}
	return intSlice, nil
}

// Основная функция для сортировки и удаления дубликатов
func StringSliceToIntSortAndRemoveDuplicates(stringSlice []string) ([]int, error) {
	// Преобразование строк в целые числа
	intSlice, err := StringToInt(stringSlice)
	if err != nil {
		return nil, err
	}

	// Сортировка целых чисел
	sort.Ints(intSlice)

	// Преобразование обратно в строки
	uniqueStrings := removeDuplicates(intSlice)

	return uniqueStrings, nil
}

// Функция для удаления дубликатов из отсортированного слайса
func removeDuplicates(intSlice []int) []int {
	if len(intSlice) <= 1 {
		return intSlice
	}

	result := make([]int, 0, len(intSlice))
	result = append(result, intSlice[0])
	for i := 1; i < len(intSlice); i++ {
		if intSlice[i] != result[len(result)-1] {
			result = append(result, intSlice[i])
		}
	}
	return result
}

func FindMinDate(dates []time.Time) time.Time {
	minDate := dates[0]
	for _, date := range dates {
		if date.Before(minDate) {
			minDate = date
		}
	}
	return minDate
}

func GetClosestWeekday(targetWeekday int, currentDate time.Time) time.Time {
	currentWeekday := int(currentDate.Weekday())

	// Рассчитываем смещение до нужного дня недели
	offset := targetWeekday - currentWeekday
	if offset <= 0 {
		offset += 7
	}

	// Добавляем смещение к текущей дате
	closestDate := currentDate.AddDate(0, 0, offset)
	return closestDate
}

func GetClosesDateOfMonth(day int, currentMonth int, date time.Time) time.Time {

	currentYear := date.Year()

	if day > 0 {
		for {
			lastDayOfMonth := time.Date(currentYear, (time.Month(currentMonth))+1, 0, 0, 0, 0, 0, time.UTC).Day()
			nextDate := time.Date(currentYear, time.Month(currentMonth), day, 0, 0, 0, 0, time.UTC)
			if nextDate.After(date) && lastDayOfMonth >= day {
				return nextDate
			}
			currentMonth++
			if currentMonth > 12 {
				currentMonth = 1
				currentYear++
			}
		}
	}

	// Если day отрицательный, находим последний день месяца
	lastDayOfMonth := time.Date(currentYear, time.Month(currentMonth+1), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Day()
	targetDay := lastDayOfMonth + day + 1

	fmt.Println(time.Date(currentYear, time.Month(currentMonth), targetDay, 0, 0, 0, 0, time.UTC))

	return time.Date(currentYear, time.Month(currentMonth), targetDay, 0, 0, 0, 0, time.UTC)
}

func GetDateOfMonth(day int, currentMonth int, nowdate time.Time, date time.Time) time.Time {

	currentYear := date.Year()

	if day > 0 {
		for {
			lastDayOfMonth := time.Date(currentYear, (time.Month(currentMonth))+1, 0, 0, 0, 0, 0, time.UTC).Day()
			nextDate := time.Date(currentYear, time.Month(currentMonth), day, 0, 0, 0, 0, time.UTC)
			if nextDate.After(date) && lastDayOfMonth >= day && nextDate.After(nowdate) {
				return nextDate
			}
			currentYear++
		}
	}

	// Если day отрицательный, находим последний день месяца
	lastDayOfMonth := time.Date(currentYear, time.Month(currentMonth+1), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Day()
	targetDay := lastDayOfMonth + day + 1

	fmt.Println(time.Date(currentYear, time.Month(currentMonth), targetDay, 0, 0, 0, 0, time.UTC))

	return time.Date(currentYear, time.Month(currentMonth), targetDay, 0, 0, 0, 0, time.UTC)
}
