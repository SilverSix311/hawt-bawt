package palworld

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

type player struct {
	Name      string `csv:"name"`
	PlayeruId string `csv:"playeruid"`
	SteamId   string `csv:"steamid"`
}

var (
	players = []*player{}

	currentPlayerCountMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "palworld_player_count",
		Help: "The current number of players",
	})

	infoCommand = &discordgo.ApplicationCommand{
		Name:        "info",
		Description: "Get server info",
	}
	playersCommand = &discordgo.ApplicationCommand{
		Name:        "players",
		Description: "Get players",
	}
	broadcastCommand = &discordgo.ApplicationCommand{
		Name:        "broadcast",
		Description: "Broadcast a message",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "broadcast-message",
				Description: "Broadcast message",
				Required:    true,
			},
		},
	}
	saveCommand = &discordgo.ApplicationCommand{
		Name:        "save",
		Description: "Save the server",
	}

	commandHandlers = map[*discordgo.ApplicationCommand]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		infoCommand:      basicCommandHandler,
		playersCommand:   playersCommandHandler,
		broadcastCommand: basicCommandHandler,
		saveCommand:      basicCommandHandler,
	}
)

func Initialize(s *discordgo.Session) (map[*discordgo.ApplicationCommand]func(s *discordgo.Session, i *discordgo.InteractionCreate), error) {
	// Start timers
	go refreshPlayers(s, 10*time.Second)

	return commandHandlers, nil
}

func basicCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	command := ""

	switch i.ApplicationCommandData().Name {
	case infoCommand.Name:
		command = "info"
	case broadcastCommand.Name:
		command = fmt.Sprintf("broadcast %s", i.ApplicationCommandData().Options[0].StringValue())
	case saveCommand.Name:
		command = "save"
	}

	// Execute RCON command
	info, err := rconCommand(command)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error executing command: " + err.Error(),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Respond to the command
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: info,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func playersCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Respond to the command
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title: "Players",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:  "Count",
							Value: fmt.Sprintf("%d", len(players)),
						},
						{
							Name: "Players",
							Value: func() string {
								if len(players) > 0 {
									playerNames := ""
									for i := 0; i < len(players); i++ {
										playerNames += players[i].Name
										if i != len(players) {
											playerNames += "\n"
										}
									}
									return playerNames
								}
								return "None"
							}(),
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func rconCommand(command string) (string, error) {
	log.Println("Executing command: " + command)

	w := bytes.Buffer{}

	exec := executor.NewExecutor(nil, &w, "")
	defer exec.Close()

	if err := exec.Execute(&w, &config.Session{
		Address:  os.Getenv("RCON_HOST"),
		Password: os.Getenv("RCON_PASSWORD"),
	},
		command); err != nil {
		return "", err
	}

	return w.String(), nil
}

func refreshPlayers(s *discordgo.Session, interval time.Duration) {
	initial := true

	for {
		// Call RCON to get players
		playersString, err := rconCommand("showplayers")
		if err != nil {
			log.Println("Error getting players: " + err.Error())
			continue
		}

		// Parse players CSV
		newPlayers := []*player{}
		err = gocsv.Unmarshal(bytes.NewBuffer([]byte(playersString)), &newPlayers)
		if err != nil {
			log.Println("Error getting players: " + err.Error())
			continue
		}

		if !initial {
			// Check for players joining
			for _, newPlayer := range newPlayers {
				exists := false
				for _, player := range players {
					if newPlayer.SteamId == player.SteamId {
						exists = true
						continue
					}
				}
				if !exists {
					// Player joined
					_, err = s.ChannelMessageSend(os.Getenv("CHANNEL_ID"), fmt.Sprintf("%s has joined the server", newPlayer.Name))
					if err != nil {
						log.Println("Error sending players update message: " + err.Error())
						continue
					}
				}
			}

			// Check for players leaving
			for _, player := range players {
				exists := false
				for _, newPlayer := range newPlayers {
					if newPlayer.SteamId == player.SteamId {
						exists = true
						continue
					}
				}
				if !exists {
					// Player left
					_, err = s.ChannelMessageSend(os.Getenv("CHANNEL_ID"), fmt.Sprintf("%s has left the server", player.Name))
					if err != nil {
						log.Println("Error sending players update message: " + err.Error())
						continue
					}
				}
			}
		} else {
			initial = false
		}

		players = newPlayers

		// Update metrics
		currentPlayerCountMetric.Set(float64(len(players)))

		time.Sleep(interval)
	}
}
