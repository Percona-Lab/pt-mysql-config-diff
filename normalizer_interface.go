package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type normalizer func(interface{}) interface{}
type normalizers []normalizer

func Normalize(str interface{}) interface{} {
	normalizers := normalizers{
		sizesNormalizer,
		numbersNormalizer,
		setsNormalizer,
	}
	for _, normalizer := range normalizers {
		str = normalizer(str)
	}

	return str
}

func sizesNormalizer(value interface{}) interface{} {
	re := regexp.MustCompile(`(?i)^(\d*?)([KMGT])$`)

	replaceMap := map[string]int64{
		"K": 1024,
		"M": 1048576,
		"G": 1073741824,
		"T": 1099511627776,
	}
	if groups := re.FindStringSubmatch(fmt.Sprintf("%s", value)); len(groups) > 0 {
		numPart := groups[1]
		multiplier := replaceMap[strings.ToUpper(groups[2])]
		i, _ := strconv.ParseInt(numPart, 10, 64)

		return fmt.Sprintf("%d", i*multiplier)
	}

	return value
}

func numbersNormalizer(value interface{}) interface{} {
	float1, err := strconv.ParseFloat(fmt.Sprintf("%s", value), 64)
	if err == nil {
		return fmt.Sprintf("%.0f", float1)
	}
	return value
}

func setsNormalizer(value interface{}) interface{} {
	splitedValues := strings.Split(fmt.Sprintf("%s", value), ",")
	sort.Strings(splitedValues)

	return strings.Join(splitedValues, ",")
}
