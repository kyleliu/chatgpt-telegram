package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kyleliu/chatgpt-telegram/src/chatgpt"
	"github.com/kyleliu/chatgpt-telegram/src/config"
	"github.com/kyleliu/chatgpt-telegram/src/tgbot"
	"github.com/sirupsen/logrus"
)

func main() {
	// 初始化logrus日志
	log := logrus.New()
	log.Formatter = &logrus.JSONFormatter{}
	log.Out = os.Stdout
	file, err := os.OpenFile("bot.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.WithError(err).Error("Failed to open log file")
	}

	// 从.env文件中加载配置
	envConfig, err := config.LoadEnvConfig(".env")
	if err != nil {
		log.Fatalf("Couldn't load .env config: %v", err)
	}

	// 验证配置是否有效
	if err := envConfig.ValidateWithDefaults(); err != nil {
		log.Fatalf("Invalid .env config: %v", err)
	}

	// 初始化ChatGPT服务
	chatGPT := chatgpt.Init(envConfig.OpenAIKey, envConfig.PromptInit, log)
	log.Println("Started ChatGPT")

	// 初始化Telegram bot服务
	bot, err := tgbot.New(envConfig.TelegramToken, time.Duration(envConfig.EditWaitSeconds*int(time.Second)))
	if err != nil {
		log.Fatalf("Couldn't start Telegram bot: %v", err)
	}

	// 注册信号量通知中断
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		bot.Stop()
		os.Exit(0)
	}()

	// 设置日志记录器
	log.Printf("Started Telegram bot! Message @%s to start.", bot.Username)
	logger := log.WithFields(logrus.Fields{
		"bot_username": bot.Username,
		"bot_id":       bot.ID,
	})

	// 接收更新消息并处理
	for update := range bot.GetUpdatesChan() {
		if update.Message == nil {
			continue
		}

		var (
			updateText      = update.Message.Text
			updateChatID    = update.Message.Chat.ID
			updateMessageID = update.Message.MessageID
			updateUserID    = update.Message.From.ID
			updateUserName  = update.Message.From.UserName
			gptChatID       = fmt.Sprintf("%v:%v", updateChatID, updateUserID)
		)

		// 记录接收到的消息
		logger.WithFields(logrus.Fields{
			"chat_id":    updateChatID,
			"user_id":    updateUserID,
			"user_name":  updateUserName,
			"message_id": updateMessageID,
			"text":       updateText,
		}).Info("Received message")

		// 在群组中忽略非针对bot的消息
		if update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup() {
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

		// 是私人消息则不需要提到它。
		if update.Message.Chat.Type == "private" {
			updateMessageID = 0
		}

		// 检查用户是否被授权
		if len(envConfig.TelegramID) != 0 && !envConfig.HasTelegramID(updateUserID) {
			log.Printf("User %d is not allowed to use this bot", updateUserID)
			bot.Send(updateChatID, updateMessageID, "You are not authorized to use this bot.")
			continue
		}

		// 如果不是命令，则发送输入到ChatGPT，并发送回复
		if !update.Message.IsCommand() {
			bot.SendTyping(updateChatID)

			reply, err := chatGPT.SendMessage(gptChatID, updateText)
			if err != nil {
				reply = &chatgpt.ChatResponse{
					Message: fmt.Sprintf("出错了: %v", err),
				}
				bot.Send(updateChatID, updateMessageID, reply.Message)
			} else {
				bot.SendAsLiveOutput(updateChatID, updateMessageID, reply)
			}

			// 记录发送的消息
			logger.WithFields(logrus.Fields{
				"chat_id":    updateChatID,
				"user_id":    updateUserID,
				"user_name":  updateUserName,
				"message_id": updateMessageID,
				"text":       reply.Message,
			}).Info("Sent message")
			continue
		}

		// 如果是命令，则根据不同的命令执行相应的操作
		var text string
		switch update.Message.Command() {
		case "help":
			text = "发送一条消息开始与ChatGPT交谈。您可以在任何时候使用 /reload 清除对话历史并重新开始（不用担心，它不会删除Telegram消息）。"
		case "start":
			text = "发送一条消息开始与ChatGPT交谈。您可以在任何时候使用 /reload 清除对话历史并重新开始（不用担心，它不会删除Telegram消息）。"
		case "reload":
			chatGPT.ResetConversation(gptChatID, update.Message.CommandArguments())
			text = "好的，让我们重新开始吧！"
		default:
			text = "未知指令，发送 /help 获得帮助。"
		}

		if _, err := bot.Send(updateChatID, updateMessageID, text); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	}
}
