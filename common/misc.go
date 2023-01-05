package common

// SliceHasString - function to check whether a given string exists in a given slice of strings
func SliceHasString(slice []string, str string) bool {
    for _, s := range slice {
        if str == s {
            return true
        }
    }
    return false
}
