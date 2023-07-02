package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
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
const mainDir = "/app"
const videosDir = "/app/videos"
const cookiesFile = "/app/cookies.txt"

func main() {

	// Get DISCORD_API environment variable
	apiKey := os.Getenv("DISCORD_API")
	if apiKey == "" {
		log.Fatalln("Could not retrieve API token from environment variable")
	}

	// Create Discord bot session
	dg, err := discordgo.New("Bot " + apiKey)
	if err != nil {
		log.Fatalln("Could not create Discord session: ", err)
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
		log.Fatalln("Error opening Discord websocket: ", err)
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

// Function called when messages new messages are detected
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore messages created by the bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Message content
	content := m.Content

	// Message reference
	msgRef := m.Reference()

	// Check if the message has an Instagram or tiktok URL
	if !urlCheck(content) {
		return
	}

	log.Printf("Video URL detected in '%s'", content)

	// Extracting URLs from message
	urls := urlExtract(content)
	if len(urls) == 0 {
		log.Println("No URLs extracted")
		return
	}

	// Download and send video running as goroutine
	log.Printf("Download and send videos running as goroutine")
	go sendVideo(urls, s, m, msgRef)
}

// Function for checking if URL is from one of the supported sites
func urlCheck(content string) bool {
	domains := []string{"instagram.com/reel", "tiktok.com", "youtube.com/shorts"}
	for _, domain := range domains {
		if strings.Contains(content, domain) {
			return true
		}
	}

	return false
}

// Function for extracting URL from messages
func urlExtract(msg string) []string {
	// Regex for finding URL substrings in string
	//re := regexp.MustCompile("(?i)\b((?:https?://|www\\d{0,3}[.]|[a-z0-9.\\-]+[.][a-z]{2,4}/)(?:[^\\s()<>]+|\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\))+(?:\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\)|[^\\s`!()\\[\\]{};:'\\\".,<>?«»“”‘’]))")
	//re := regexp.MustCompile(`((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[.\!\/\\w]*))?)`)
	re := regexp.MustCompile(`((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w\-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[.\!\/\\\w]*))?)`)

	// Checking the msg string for URLs using the re regex
	urls := re.FindAllString(msg, -1)

	log.Printf("Checked message for URLs")

	return urls
}

func downloadVideo(url string) (string, string, error) {
	// Create new directory for video
	dirName, _ := genRandomStr(10)
	vidPath := videosDir + "/" + dirName
	log.Printf("Creating directory %s", vidPath)
	err := os.Mkdir(vidPath, 700)
	if err != nil {
		log.Printf("Could not create directory %s: %s", vidPath, err)
		return "", "", errors.New("could not create directory")
	}

	// Change to videos/ directory to store videos
	err = os.Chdir(vidPath)
	if err != nil {
		log.Printf("Could not change directory to %s: %s", vidPath, err)
		return "", "", errors.New("failed changing directory")
	}

	// Check if TikTok URL is for a video
	if strings.Contains(url, "tiktok") {
		if !strings.Contains(url, "vm.tiktok") && !strings.Contains(url, "/@") {
			return "", "", errors.New("not URL for a TikTok video")
		} else {
			// Get the final URL after redirects
			log.Printf("Following redirect from %s", url)
			url, err = followRedir(url)
			if err != nil {
				log.Fatalf("Following redirect failed")
				// return "", "", errors.New("following redirect failed")
			}
			log.Printf("Followed redirect until %s", url)
		}
	}

	// Create command to download video using yt-dlp
	log.Printf("Downloading: %s", url)

	// Variable for the argument passing the cookies.txt file
	// cookiesArg := fmt.Sprintf("'--cookies %s'", cookiesFile)
	// ytdlpArgs := fmt.Sprintf("-c -p %s", url)
	cmd := exec.Command(ytdlp, "-p", "'--cookies", "/app/cookies.txt'", "-c", url)
	//cmd := exec.Command("/bin/bash", "-c" ytdlp, "-p", cookiesArg, "-c", url)

	log.Printf("Command: %s", cmd.String())

	// Stream output from cmd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Printf("Could not execute ytdlp: %s", err)
	}

	// Get name of downloaded video
	videos, err := ioutil.ReadDir(vidPath)
	if err != nil {
		log.Printf("Could not list files in %s: %s", vidPath, err)
		return "", "", errors.New("could not list files in directory")
	}
	video := videos[0].Name()

	// Change back to the main working directory
	err = os.Chdir(mainDir)
	if err != nil {
		log.Printf("Could not change directory to %s: %s", mainDir, err)
		return "", "", errors.New("failed changing directory")
	}

	return video, vidPath, nil
}

func sendVideo(urls []string, s *discordgo.Session, m *discordgo.MessageCreate, msgRef *discordgo.MessageReference) {
	log.Print("Downloading and sending videos.")
	for _, url := range urls {

		var vidPath string // Variable for storing path of downloaded video
		var video string   // Variable for storing name of downloaded video
		var err error      // Variable for storing error code

		maxAttempts := 3 // Maximum amount of allowed attempts
		attempt := 0     // Variable for current attempt number

		for {
			log.Printf("Downloading: %s", url)
			video, vidPath, err = downloadVideo(url)
			if err != nil {
				log.Printf("Could not download %s: %s", video, err)
				attempt += 1

				if attempt >= maxAttempts {
					break
				} else {
					continue
				}
			}

			// If the downloaded file is called .tmp the download has failed. Retry
			if video == ".tmp" {
				attempt += 1
				log.Printf("Failed to download video. Retries left: %d", maxAttempts-attempt)

				// Delete failed downloaded video and its directory
				log.Printf("Deleting %s/%s", vidPath, video)
				err = os.RemoveAll(vidPath + "/")
				if err != nil {
					log.Printf("Failed to remove %s: %s", vidPath, err)
				}

				// Exit if max amount of attempts is reached
				if attempt >= maxAttempts {
					log.Printf("Could not download video after %d attempts. Exiting...", attempt)
					return
				} else {
					continue
				}

			} else {
				break
			}
		}

		// Opening video file for reading
		log.Printf("Opening %s", video)
		vidReader, err := os.Open(vidPath + "/" + video)
		if err != nil {
			log.Printf("Failed to open %s: %s", video, err)
			continue
		}

		// Constructing discordgo.File object of the downloaded videofile
		log.Printf("Constructing discordgo.File object and pointer")
		vidFile := []*discordgo.File{{Name: video, ContentType: "video/mp4", Reader: vidReader}}

		// Constructing discordgo.MessageSend object to send video
		log.Printf("Constructing discordgo.MessageSend object")
		msgSend := &discordgo.MessageSend{Files: vidFile, Reference: msgRef}

		// Sending video
		log.Printf("Sending video: %s", video)
		_, err = s.ChannelMessageSendComplex(m.ChannelID, msgSend)

		// Closing video file
		log.Printf("Closing %s", video)
		err = vidReader.Close()
		if err != nil {
			log.Printf("Could not close %s: %s", video, err)
		}

		// Delete video and its directory
		log.Printf("Deleting %s/%s", vidPath, video)
		err = os.RemoveAll(vidPath + "/")
		if err != nil {
			log.Printf("Failed to remove %s: %s", vidPath, err)
		}
	}
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

// Generate random string
// Used for unique directory names for videos
// SRC: https://stackoverflow.com/a/22892986/11234304
func genRandomStr(strLen int) (string, error) {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, strLen)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:strLen], nil
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
