package jukebox

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gocarina/gocsv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/thegrandpackard/rcon-cli/library/config"
	"github.com/thegrandpackard/rcon-cli/library/executor"
)

var (
	// Define Discord application commands
	playCommand = &discordgo.ApplicationCommand{
		Name:        "play",
		Description: "Play a song in the voice channel using URL",
	}

	// Map to store command handlers for Discord application commands
	commandHandlers = map[*discordgo.ApplicationCommand]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		playCommand:   basicCommandHandler,
	}
)

// Initialize the Jukebox module
func Initialize(s *discordgo.Session) (map[*discordgo.ApplicationCommand]func(s *discordgo.Session, i *discordgo.InteractionCreate), error) {
	return commandHandlers, nil
}

// Handler for basic Discord application commands
func basicCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	command := ""

	switch i.ApplicationCommandData().Name {
	case playCommand.Name:
		command = "play"
	}

	// Respond to the command with the result
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: info,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// Play a song in the voice channel using URL from the user
func Play(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get the voice channel the user is in
	vc, err := s.ChannelVoiceJoin(i.GuildID, i.Member.Voice.ChannelID, false, true)
	if err != nil {
		log.Println("Error joining voice channel: ", err)
		return
	}

	// Get the URL from the user
	url := i.ApplicationCommandData().Options[0].StringValue()

	// Play the song in the voice channel
	play(vc, url)
}