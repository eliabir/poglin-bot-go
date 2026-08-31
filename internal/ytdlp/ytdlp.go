package ytdlp

import (
	"os"
	"io"
	"fmt"
	"log"
	"errors"
	"bytes"
	"strings"
	"os/exec"
)

const ytdlp = "/app/yt-dlp_discord"
const cookiesFile = "/app/cookies.txt"
const useCookies = false

// Indicates that the URL is unsupported by yt-dlp
var ErrUnsupportedUrl = errors.New("unsupported url")

func Run(url string) error {
	var ytdlpArgs string
	var cmd *exec.Cmd
	if useCookies {
		cookiesArg := fmt.Sprintf("\"--cookies %s\"", cookiesFile) // Variable for the argument passing the cookies.txt file
		ytdlpArgs = fmt.Sprintf("%s -c -p %s %s", ytdlp, cookiesArg, url)

		// Execute command to download video using yt-dlp_discord
		log.Printf("Downloading: %s", url)
		cmd = exec.Command("/bin/bash", "-c", ytdlp, "-c", "-p", cookiesArg, url)
	} else {
		ytdlpArgs = fmt.Sprintf("%s -c %s", ytdlp, url)
		cmd = exec.Command("/bin/bash", "-c", ytdlpArgs)
	}

	log.Printf("Command: %s", cmd.String())

	// Stream and store output from cmd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)

	// Run ytdlp command
	err := cmd.Run()
	if err != nil {
		outputErr := stderr.String()
		log.Printf("Could not execute ytdlp: %s", outputErr)
		if strings.Contains(outputErr, "Unsupported URL") {
			return fmt.Errorf("download failed: %w", ErrUnsupportedUrl)
		}
		return fmt.Errorf("download failed: %w", err)
	}
	return nil
}
