package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	BASEURL = "https://api.openai.com/v1"
)

var messages []Message
var usersMessage map[int64][]Message

// need set messages to the users'id
func main() {
	// set log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		panic(err)
	}
	messages = make([]Message, 0)
	usersMessage = make(map[int64][]Message)
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
}

func addMessageToHistory(message Message, userID int64) {
	messages = usersMessage[userID]
	messages = append(messages, message)
	usersMessage[userID] = messages
}

func clearUserChatHistory(userID int64) {
	messages = make([]Message, 0)
	usersMessage[userID] = messages
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
	// add user's message to history
	message := Message{
		Role:    "user",
		Content: text,
	}
	addMessageToHistory(message, userID)
	response := sendMsgToChatGpt(messages)
	// add assistant's message to history
	message = Message{
		Role:    "assistant",
		Content: response,
	}
	addMessageToHistory(message, userID)
	log.Println(usersMessage)
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
