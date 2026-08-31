package video

import (
	"os"
	"log"
	"fmt"
	"errors"
	"strings"

	"github.com/eliabir/poglin-bot-go/internal/ytdlp"

	"github.com/google/uuid"
	"github.com/bwmarrin/discordgo"
)

const mainDir = "/app"
const videosDir = "/app/videos"
const downloadRetries = 5

var ErrNotTiktokURL = errors.New("not url for a tiktok video")
var ErrDirEmpty = errors.New("directory empty")

type Video struct {
	path string
	name string
}

func download(url string) (*Video, error) {
	// Create new directory for video
	dirName := uuid.New().String()
	videoPath := videosDir + "/" + dirName

	log.Printf("Creating directory %s", videoPath)
	err := os.Mkdir(videoPath, 0700)
	if err != nil {
		return nil, fmt.Errorf("creating directory: %w", err)
	}

	// Change to videos/ directory to store videos
	err = os.Chdir(videoPath)
	if err != nil {
		return nil, fmt.Errorf("changing directory to %s: %w", videoPath, err)
	}

	// Check if TikTok URL is for a video
	if strings.Contains(url, "tiktok") {
		if !strings.Contains(url, "vm.tiktok") && !strings.Contains(url, "/@") {
			return nil, fmt.Errorf("downloading video failed: %w", ErrNotTiktokURL)
		}
	}

	log.Printf("Downloading video...")
	if err = ytdlp.Run(url); err != nil {
		return nil, fmt.Errorf("running yt-dlp: %w", err)
	}

	// Get name of downloaded video
	videos, err := os.ReadDir(videoPath)
	if err != nil {
		return nil, fmt.Errorf("listing files in %s: %w", videoPath, err)
	}

	// Check if a video was actually downloaded
	if len(videos) == 0 {
		log.Printf("No videos found in %s", videoPath)
		return nil, fmt.Errorf("no videos in %s: %w", videoPath, ErrDirEmpty)
	}

	videoName := videos[0].Name()

	// Change back to the main working directory
	err = os.Chdir(mainDir)
	if err != nil {
		return nil, fmt.Errorf("changing directory to %s: %w", mainDir, err)
	}

	return &Video{path: videoPath, name: videoName}, nil
}

func Handle(urls []string, s *discordgo.Session, m *discordgo.MessageCreate, msgRef *discordgo.MessageReference) {
	for _, url := range urls {

		var video *Video
		var err error

		maxAttempts := downloadRetries // Maximum amount of allowed attempts
		attempt := 0                   // Variable for current attempt number

		for {
			log.Printf("Downloading: %s", url)
			video, err = download(url)
			if err != nil {
				log.Printf("Downloading video failed: %v", err)

				if errors.Is(err, ytdlp.ErrUnsupportedUrl) {
					log.Printf("Unsupported url %s: %v", url, err)
					break
				}

				attempt += 1

				if attempt >= maxAttempts {
					log.Printf("Exhausted all download attempts trying to download %s", url)
					break
				}

				continue
			}

			// If the downloaded file is called .tmp the download has failed. Retry
			if video.name == ".tmp" {
				attempt += 1
				log.Printf("Failed to download video. Retries left: %d", maxAttempts-attempt)

				// Delete failed downloaded video and its directory
				log.Printf("Deleting %s/%s", video.path, video.name)
				err = os.RemoveAll(video.path + "/")
				if err != nil {
					log.Printf("Failed to remove %s: %v", video.path, err)
				}

				if attempt >= maxAttempts {
					log.Printf("Could not download video after %d attempts", attempt)
					break
				}

				continue

			} else if video.name == "" {
				log.Printf("No video downloaded")

				if attempt >= maxAttempts {
					log.Printf("Could not download video after %d attempts", attempt)
					break
				} else {
					continue
				}

			} else {
				// Download seems to be successful
				break
			}
		}

		if video == nil {
			continue
		}

		// Opening video file for reading
		log.Printf("Opening %s", video.name)
		videoFile, err := os.Open(video.path + "/" + video.name)
		if err != nil {
			log.Printf("Failed to open %s: %s", video.name, err)
			continue
		}

		// Constructing discordgo.File object of the downloaded videofile
		log.Printf("Constructing discordgo.File object and pointer")
		discordFile := []*discordgo.File{{Name: video.name, ContentType: "video/mp4", Reader: videoFile}}

		// Constructing discordgo.MessageSend object to send video
		log.Printf("Constructing discordgo.MessageSend object")
		message := &discordgo.MessageSend{Files: discordFile, Reference: msgRef}

		// Sending video
		log.Printf("Sending video: %s", video.name)
		_, err = s.ChannelMessageSendComplex(m.ChannelID, message)
		if err != nil {
			log.Printf("sending video %s failed: %v", video.name, err)
			continue
		}

		// Closing video file
		log.Printf("Closing %s", video)
		err = videoFile.Close()
		if err != nil {
			log.Printf("closing %s failed: %v", video, err)
			continue
		}

		// Delete video and its directory
		log.Printf("Deleting %s/%s", video.path, video)
		err = os.RemoveAll(video.path + "/")
		if err != nil {
			log.Printf("failed to remove %s: %v", video.path, err)
			continue
		}
	}
}
