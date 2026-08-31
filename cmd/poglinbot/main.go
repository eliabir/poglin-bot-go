package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/eliabir/poglin-bot-go/internal/config"
	"github.com/eliabir/poglin-bot-go/internal/url"
	"github.com/eliabir/poglin-bot-go/internal/video"

	"github.com/bwmarrin/discordgo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	// Create Discord bot session
	dg, err := discordgo.New("Bot " + cfg.ApiKey)
	if err != nil {
		log.Fatalf("starting discord bot session: %v", err)
	}

	// Add ready() function as callback for ready events
	dg.AddHandler(ready)

	// Add messageCreate() function as a callback for messageCreate events
	dg.AddHandler(messageCreate)

	// Store information about guilds, messages and voice states
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	// Open websocket and wait for termination signal
	err = dg.Open()
	if err != nil {
		log.Fatalf("opening discord websocket failed: %v", err)
	}

	log.Printf("Poglin-Bot is now running. Press CTRL-C to exit.")
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
	if !url.Check(content) {
		return
	}

	log.Printf("Video URL detected in '%s'", content)

	// Extracting URLs from message
	log.Printf("Extract URLs")
	urls := url.Extract(content)
	if len(urls) == 0 {
		log.Println("No URLs extracted")
		return
	}

	// Download and send video running as goroutine
	log.Printf("Download and send videos running as goroutine")
	go video.Handle(urls, s, m, msgRef)
}
