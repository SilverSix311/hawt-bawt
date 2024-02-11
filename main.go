package main

import (
	"net/http"
	"os"
	"os/signal"

	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thegrandpackard/palworld-discord-bot/palworld"
)

var (
	commands        = []*discordgo.ApplicationCommand{}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){}
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file: " + err.Error())
	}

	// Initialize Discord Chat API
	s, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatal("Invalid bot parameters")
	}
	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) { log.Print("Bot is up!") })
	// Register the MessageComponentProcessor func as a callback for /command invocation
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if f, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			f(s, i)
		}
	})

	// Initialize modules
	palworldCommands, err := palworld.Initialize(s)
	if err != nil {
		log.Fatal("Error initializing palworld module")
	}
	for command, handler := range palworldCommands {
		commands = append(commands, command)
		commandHandlers[command.Name] = handler
	}

	// Open websocket to Discord API
	if err := s.Open(); err != nil {
		log.Fatal("Error connecting to Discord API")
	}
	defer s.Close()

	// Register slash commands
	if os.Getenv("REGISTER_COMMANDS") == "TRUE" {
		registerSlashCommands(s)
	}

	// Initialize prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)

	// Capture Ctrl-c to shut down bot
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	log.Print("Gracefully shutting down")
}

func registerSlashCommands(s *discordgo.Session) {
	log.Println("Registering slash commands")

	for _, command := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, os.Getenv("GUILD_ID"), command)
		if err != nil {
			log.Printf("Cannot create '%v' command: %v", command.Name, err)
		}
		log.Printf("Registered Slash Command: %s", command.Name)
	}
}
