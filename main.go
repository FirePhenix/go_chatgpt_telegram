package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BASEURL = "https://api.openai.com/v1"
)

// need set messages to the users'id
func main() {
	// set log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	InitDB()
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		panic(err)
	}
	bot.Debug = true
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := bot.GetUpdatesChan(updateConfig)
	for update := range updates {
		if update.Message == nil {
			continue
		}
		go handleUpdate(bot, update)
	}
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-s
		if err = client.Disconnect(context.TODO()); err != nil {
			log.Println(err)
		}
		log.Printf("Exiting...\n")
		os.Exit(0)
	}()
}

func addMessageToHistory(message Message) {
	writeMessageToDB(message)
}

func clearUserChatHistory(userID int64) {
	removeMessagesByUserID(userID)
}

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	text := update.Message.Text
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	replyMsg := tgbotapi.NewMessage(chatID, text)
	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "new":
			replyMsg.Text = "ok, let's start a new conversation"
			clearUserChatHistory(userID)
		case "history":
			replyMsg.Text = "coming soon"
		default:
			replyMsg.Text = "I don't know that command"
		}
	}
	typingAction := tgbotapi.NewChatAction(chatID, "typing")
	_, _ = bot.Send(typingAction)
	message := Message{
		Role:    "user",
		Content: text,
		UserID:  userID,
	}
	addMessageToHistory(message)
	messages := getMessagesByUserID(userID)
	log.Println(messages)
	response := sendMsgToChatGpt(messages)
	// add assistant's message to history
	message = Message{
		Role:    "assistant",
		Content: response,
		UserID:  userID,
	}
	addMessageToHistory(message)
	replyMsg.Text = response
	_, _ = bot.Send(replyMsg)
}

func sendMsgToChatGpt(messages []Message) string {
	parameters := map[string]interface{}{
		"model":    "gpt-3.5-turbo",
		"messages": messages,
	}
	body := new(bytes.Buffer)
	json.NewEncoder(body).Encode(parameters)
	url := BASEURL + "/chat/completions"
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_KEY"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Println(string(response))
	var chatResponse ChatResponse
	err = json.Unmarshal(response, &chatResponse)
	if err != nil {
		panic(err)
	}
	content := chatResponse.Choices[0].Message.Content

	return content
}
