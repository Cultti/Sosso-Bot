package discord

import (
	"sort"
	"strconv"
	"strings"
)

func sortChampionships(list []string) []string {
	// Separate Mestaruussarja and others
	var mestaruussarja []string
	var others []string

	for _, val := range list {
		if strings.HasPrefix(val, "Mestaruussarja") {
			mestaruussarja = append(mestaruussarja, val)
		} else {
			others = append(others, val)
		}
	}

	// Sort others numerically by the first number in the string
	sort.SliceStable(others, func(i, j int) bool {
		numI := extractNumber(others[i])
		numJ := extractNumber(others[j])
		return numI < numJ
	})

	// Return combined list
	return append(mestaruussarja, others...)
}

// Extract the first number in the string (e.g., "24 Divisioona S11" -> 24)
func extractNumber(s string) int {
	parts := strings.Fields(s)
	for _, p := range parts {
		if n, err := strconv.Atoi(p); err == nil {
			return n
		}
	}
	return 0
}
