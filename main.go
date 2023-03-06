package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kyleliu/chatgpt-telegram/src/chatgpt"
	"github.com/kyleliu/chatgpt-telegram/src/config"
	"github.com/kyleliu/chatgpt-telegram/src/tgbot"
)

func main() {
	envConfig, err := config.LoadEnvConfig(".env")
	if err != nil {
		log.Fatalf("Couldn't load .env config: %v", err)
	}
	if err := envConfig.ValidateWithDefaults(); err != nil {
		log.Fatalf("Invalid .env config: %v", err)
	}

	chatGPT := chatgpt.Init(envConfig.OpenAIKey)
	log.Println("Started ChatGPT")

	bot, err := tgbot.New(envConfig.TelegramToken, time.Duration(envConfig.EditWaitSeconds*int(time.Second)))
	if err != nil {
		log.Fatalf("Couldn't start Telegram bot: %v", err)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		bot.Stop()
		os.Exit(0)
	}()

	log.Printf("Started Telegram bot! Message @%s to start.", bot.Username)

	for update := range bot.GetUpdatesChan() {
		if update.Message == nil {
			continue
		}

		var (
			updateText      = update.Message.Text
			updateChatID    = update.Message.Chat.ID
			updateMessageID = update.Message.MessageID
			updateUserID    = update.Message.From.ID
		)

		// ignore messages in group if it's not mention to me
		if update.Message.Chat.Type == "group" || update.Message.Chat.Type == "supergroup" {
			mentioned := false
			newText := updateText
			for _, entity := range update.Message.Entities {
				if entity.Type != "mention" {
					continue
				}

				mention := "@" + bot.Username
				newText = strings.ReplaceAll(newText, mention, "")
				if strings.Contains(updateText, mention) {
					mentioned = true
				}

				log.Print("=> mention:", mention)
			}
			if !mentioned {
				continue
			}
			updateText = newText
		}

		// is private message? don't mention it.
		if update.Message.Chat.Type == "private" {
			updateMessageID = 0
		}

		if len(envConfig.TelegramID) != 0 && !envConfig.HasTelegramID(updateUserID) {
			log.Printf("User %d is not allowed to use this bot", updateUserID)
			bot.Send(updateChatID, updateMessageID, "You are not authorized to use this bot.")
			continue
		}

		if !update.Message.IsCommand() {
			bot.SendTyping(updateChatID)

			reply, err := chatGPT.SendMessage(updateText, updateChatID)
			if err != nil {
				bot.Send(updateChatID, updateMessageID, fmt.Sprintf("Error: %v", err))
			} else {
				bot.SendAsLiveOutput(updateChatID, updateMessageID, reply)
			}
			continue
		}

		var text string
		switch update.Message.Command() {
		case "help":
			text = "Send a message to start talking with ChatGPT. You can use /reload at any point to clear the conversation history and start from scratch (don't worry, it won't delete the Telegram messages)."
		case "start":
			text = "Send a message to start talking with ChatGPT. You can use /reload at any point to clear the conversation history and start from scratch (don't worry, it won't delete the Telegram messages)."
		case "reload":
			chatGPT.ResetConversation(updateChatID, updateText)
			text = "Started a new conversation. Enjoy!"
		default:
			text = "Unknown command. Send /help to see a list of commands."
		}

		if _, err := bot.Send(updateChatID, updateMessageID, text); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	}
}
