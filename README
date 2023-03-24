# README.md

## Telegram Chatbot with GPT-3.5-turbo

This is a simple Telegram chatbot that uses the OpenAI GPT-3.5-turbo model to generate responses. It stores message history per user and sends it to the GPT-3.5-turbo API for context-aware conversation.

### Dependencies

- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [OpenAI GPT-3.5-turbo API](https://platform.openai.com/docs/api-reference)

### Installation

1. Clone the repository
2. Install dependencies with `go mod tidy`
3. Set the environment variables `TELEGRAM_TOKEN` and `OPENAI_KEY`
4. Run the project with `go run main.go`

### Usage

Users can interact with the chatbot in Telegram, sending messages and receiving GPT-3.5-turbo-generated replies. The chatbot also supports the following commands:

- `/new`: Starts a new conversation, clearing the previous message history for the user.
- `/history`: Currently not implemented.

### Data Structures

- `Message`: Represents a message in the conversation, with a role (user or assistant) and content.
- `ChatResponse`: Represents the GPT-3.5-turbo API response, with choices, message content, and usage information.

### Note

This is a basic implementation of a Telegram chatbot using the GPT-3.5-turbo API. 
