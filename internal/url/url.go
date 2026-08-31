package url

import (
	"regexp"
	"strings"
)

// Function for checking if URL is from one of the supported sites
func Check(content string) bool {
	domains := []string{"instagram.com/reel", "tiktok.com", "youtube.com/shorts"}
	for _, domain := range domains {
		if strings.Contains(content, domain) {
			return true
		}
	}

	return false
}

// Function for extracting URL from messages
func Extract(msg string) []string {
	// Regex for finding URL substrings in string
	re := regexp.MustCompile(`((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w\-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[.\!\/\\\w]*))?)`)

	// Checking the msg string for URLs using the re regex
	urls := re.FindAllString(msg, -1)

	return urls
}
