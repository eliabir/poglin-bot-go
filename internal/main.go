package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

const ytdlp = "/app/yt-dlp_discord"
const videosDir = "/app/videos"

func main() {

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

	log.Print("Downloading and sending videos.")
	for _, url := range urls {
		log.Printf("Downloading: %s", url)
		video, err := downloadVideo(url)
		if err != nil {
			log.Printf("Could not download %s: %s", video, err)
			continue
		}

		// Opening video file for reading
		log.Printf("Opening %s", video)
		vidReader, err := os.Open(videosDir + "/" + video)
		if err != nil {
			log.Printf("Failed to open %s: %s", video, err)
			continue
		}

		// Constructing discordgo.File object of the downloaded videofile
		log.Printf("Constructing discordgo.File object and pointer")
		vidFile := &discordgo.File{Name: video, ContentType: "video/mp4", Reader: vidReader}
		vidFilePtrSlice := []*discordgo.File{vidFile}

		// Constructing discordgo.MessageSend object to send video
		log.Printf("Constructing discordgo.MessageSend object")
		msgSend := discordgo.MessageSend{Files: vidFilePtrSlice, Reference: m.MessageReference}
		msgSendPtr := &msgSend

		// Sending video
		log.Printf("Sending video: %s", video)
		_, err = s.ChannelMessageSendComplex(m.ChannelID, msgSendPtr)

		// Closing video file
		log.Printf("Closing %s", video)
		err = vidReader.Close()
		if err != nil {
			log.Printf("Could not close %s: %s", video, err)
		}

		// Deleting video
		log.Printf("Deleting %s", video)
		err = os.Remove(videosDir + "/" + video)
		if err != nil {
			log.Printf("Failed to remove %s", err)
		}
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

func downloadVideo(origUrl string) (string, error) {
	// Store current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("Could not get current working directory")
		return "", errors.New("could not get current working directory")
	}

	// Change to videos/ directory to store videos
	err = os.Chdir(videosDir)
	if err != nil {
		log.Printf("Could not change directory to %s: %s", videosDir, err)
		return "", errors.New("failed changing directory")
	}

	// Get the final URL after redirects
	url, err := followRedir(origUrl)
	if err != nil {
		return "", errors.New("following redirect failed")
	}

	// Check if TikTok URL is for a video
	if strings.Contains(url, "tiktok") {
		if !strings.Contains(url, "vm.tiktok") && !strings.Contains(url, "/@") {
			return "", errors.New("not URL for a TikTok video")
		}
	}

	// Create command to download video using yt-dlp
	log.Printf("Downloading: %s", url)
	cmd := exec.Command("/bin/bash", ytdlp, "-v", "-c", url)

	// Stream output from cmd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Printf("Could not execute ytdlp: %s", err)
	}

	video, err := findRecentFile(videosDir)
	if err != nil {
		log.Printf("Finding most recent file failed: %s", err)
		return "", errors.New("finding most recent file failed")
	}

	// Change back to the original working directory
	err = os.Chdir(cwd)
	if err != nil {
		log.Printf("Could not change directory to %s: %s", cwd, err)
		return "", errors.New("failed changing directory")
	}

	return video, nil
}

// Function for following URL redirects and find final URL
func followRedir(url string) (string, error) {
	// Request URL to follow redirects and find final URL
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Request to %s failed with error: %s", url, err)
		return "", errors.New("could not reach URL")
	}

	// Store the final URL
	url = resp.Request.URL.String()

	return url, nil
}

// Function for finding most recently created file in directory
// SRC: https://stackoverflow.com/a/45579190
func findRecentFile(dir string) (string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Printf("Could not list files in %s: %s", dir, err)
		return "", errors.New("could not list files")
	}

	var modTime time.Time
	var names []string

	for _, fi := range files {
		if fi.Mode().IsRegular() {
			if !fi.ModTime().Before(modTime) {
				if fi.ModTime().After(modTime) {
					modTime = fi.ModTime()
					names = names[:0]
				}
				names = append(names, fi.Name())
			}
		}
	}
	if len(names) > 0 {
		fmt.Println(modTime, names)
	}

	return names[len(names)-1], nil
}
