package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/qiniu/audio/mp3"
	_ "github.com/qiniu/audio/ogg"
)

const (
	BASEURL = "https://api.openai.com/v1"
)

// need set messages to the users'id
func main() {
	// set log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	InitDB()
	openaiKey := os.Getenv("OPENAI_KEY")
	if openaiKey == "" {
		log.Fatal("OPENAI_KEY is not set")
	}
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal("TELEGRAM_TOKEN is not set")
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
			log.Fatal(err)
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
	text := ""
	if update.Message.Voice != nil {
		text = voiceToText(bot, update.Message.Voice.FileID)
	} else {
		text = update.Message.Text
	}
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
		log.Fatal(err)
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var chatResponse ChatResponse
	err = json.Unmarshal(response, &chatResponse)
	if err != nil {
		log.Fatal(err)
	}
	content := chatResponse.Choices[0].Message.Content

	return content
}

type TranscriptionResponse struct {
	Text string `json:"text"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

func voiceToText(bot *tgbotapi.BotAPI, fileID string) string {
	file := tgbotapi.FileConfig{
		FileID: fileID,
	}
	fileResponse, err := bot.GetFile(file)
	if err != nil {
		log.Fatal(err)
	}
	fileURL := fileResponse.Link(os.Getenv("TELEGRAM_TOKEN"))
	// download file
	log.Println(fileURL)
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Println(err)
		return ""
	}
	defer resp.Body.Close()
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "mp3", "pipe:1")
	cmd.Stdin = resp.Body

	reader, writer := io.Pipe()
	cmd.Stdout = writer

	go func() {
		defer writer.Close()
		if err := cmd.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	var reqeustBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&reqeustBody)
	fieldWriter, err := multipartWriter.CreateFormField("model")
	if err != nil {
		log.Fatal(err)
	}
	fieldWriter.Write([]byte("whisper-1"))
	fileWrite, err := multipartWriter.CreateFormFile("file", fileID+".mp3")
	if err != nil {
		log.Println(err)
		return ""
	}
	if _, err = io.Copy(fileWrite, reader); err != nil {
		log.Println(err)
		return ""
	}
	if err := multipartWriter.Close(); err != nil {
		log.Fatal(err)
	}
	url := "https://api.openai.com/v1/audio/transcriptions"
	req, _ := http.NewRequest("POST", url, &reqeustBody)
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_KEY"))
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	client := &http.Client{}
	resp, err = client.Do(req)
	if resp.StatusCode != 200 {
		var errorResponse ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		log.Println(errorResponse.Error.Message)
		return ""
	}
	if err != nil {
		log.Println(err)
		return ""
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return ""
	}
	var transcriptionResponse TranscriptionResponse
	err = json.Unmarshal(response, &transcriptionResponse)
	if err != nil {
		log.Println(err)
		return ""
	}
	log.Println(transcriptionResponse.Text)
	return transcriptionResponse.Text
}
