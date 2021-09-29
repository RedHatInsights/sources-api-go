package util

// SliceContainsString returns true if the specified target is present in the given slice.
func SliceContainsString(slice []string, target string) bool {
	for _, element := range slice {
		if element == target {
			return true
		}
	}
	return false
}
