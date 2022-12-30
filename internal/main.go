package main

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Could not load .env file", err)
	}

	apiKey := os.Getenv("DISCORD_API")

	dg, err := discordgo.New("Bot " + apiKey)
	if err != nil {
		log.Fatal("Could not create Discord session: ", err)
	}

}
