package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const ytdlp = "/usr/src/bot/bash-toolbox/yt-dlp_discord"
const videosDir = "/usr/src/bot/videos"

func main() {

	// Load environment variables from .env file
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Could not load .env file: ", err)
	// }

	// Store DISCORD_API environment variable
	apiKey := os.Getenv("DISCORD_API")

	// Create Discord bot session
	dg, err := discordgo.New("Bot " + apiKey)
	if err != nil {
		log.Fatal("Could not create Discord session: ", err)
	}

	// Add ready() function as callback for ready events
	dg.AddHandler(ready)

	// Add messageCreate() function as a callback for messageCreate events
	dg.AddHandler(messageCreate)

	// Stor information about guilds, messages and voice states
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	// Open websocket and wait for termination signal
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening Discord websocket: ", err)
	}

	log.Printf("Poglin-bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("")
	log.Println("Closing websocket")

	// Close down Discord websocket
	dg.Close()
}

// Function called when bot is ready
func ready(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Logged in as %s", s.State.User.Username)
	s.UpdateGameStatus(0, "Waiting for videos")
}

// Function called when messages are created
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore messages created by the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Message content
	content := m.Content

	// Check if the message has an instagram og tiktok URL
	if !strings.Contains(content, "instagram.com/reel") && !strings.Contains(content, "tiktok.com/") {
		return
	}

	log.Printf("Video URL detected in '%s'", m.Content)

	// Extracting URLs from message
	urls := urlExtract(content)
	if len(urls) == 0 {
		log.Println("No URLs extracted")
		return
	}

	log.Print("Extracted URLs: ")
	for i := 0; i < len(urls); i++ {
		log.Printf("%s ", urls[i])
	}

}

// Function for extracting URL from messages
func urlExtract(msg string) []string {
	// Regex for finding URL substrings in string
	//re := regexp.MustCompile("(?i)\b((?:https?://|www\\d{0,3}[.]|[a-z0-9.\\-]+[.][a-z]{2,4}/)(?:[^\\s()<>]+|\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\))+(?:\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\)|[^\\s`!()\\[\\]{};:'\\\".,<>?«»“”‘’]))")
	re := regexp.MustCompile(`((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[.\!\/\\w]*))?)`)

	// Checking the msg string for URLs using the re regex
	urls := re.FindAllString(msg, -1)

	log.Printf("Checked message for URLs")

	return urls
}

func downloadVideo(urls []string) []string {
	// Store current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get current working directory")
	}

	// Change to videos/ directory to store videos
	os.Chdir(videosDir)

	for i := 0; i < len(urls); i++ {
		// Get the final URL after redirects
		url := followRedir(urls[i])
		if url == "" {
			continue
		}

		// Check if TikTok URL is for a video
		if strings.Contains(url, "tiktok") {
			if !strings.Contains(url, "vm.tiktok") && !strings.Contains(url, "/@") {
				continue
			}
		}

		// Download video from URL
		exec.Command("/bin/sh", ytdlp)
	}

	// Change back to the original working directory
	os.Chdir(cwd)

	return []string{""}
}

func followRedir(url string) string {
	// Request URL to follow redirects and find final URL
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Request to %s failed with error: %s", url, err)
		return ""
	}

	// Store the final URL
	url = resp.Request.URL.String()

	return url
}
