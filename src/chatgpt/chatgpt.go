package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

type ChatGPT struct {
	OpenAIKey     string
	Conversations map[string]*ChatGPTConversation
	PromptInit    string
	Usage         ChatGPTUsage
	Logger        *logrus.Logger
}

type ChatGPTConversation struct {
	ChatID   string
	Messages []*ChatGPTMessage
	Usage    *ChatGPTUsage
}

type ChatGPTMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatGPTRequest struct {
	Model       string            `json:"model"`
	Messages    []*ChatGPTMessage `json:"messages"`
	Temperature float32           `json:"temperature"`
}

type ChatGPTMessageChoice struct {
	Index   int `json:"index"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type ChatGPTUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatGPTResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Choices []ChatGPTMessageChoice `json:"choices"`
	Usage   ChatGPTUsage           `json:"usage"`
}

type ChatResponse struct {
	Message string
}

func Init(apiKey string, prompt string, logger *logrus.Logger) *ChatGPT {
	return &ChatGPT{
		OpenAIKey:     apiKey,
		Conversations: map[string]*ChatGPTConversation{},
		PromptInit:    prompt,
		Usage: ChatGPTUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
		Logger: logger,
	}
}

func (c *ChatGPT) ResetConversation(chatID string, basecmd string) {
	messages := []*ChatGPTMessage{}
	if basecmd == "" {
		basecmd = c.PromptInit
	}
	messages = append(messages, &ChatGPTMessage{
		Role:    "system",
		Content: basecmd,
	})

	c.Conversations[chatID] = &ChatGPTConversation{
		ChatID:   chatID,
		Messages: messages,
		Usage: &ChatGPTUsage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}
}

func (c *ChatGPT) UpdateMessage(chatID string, role string, message string) []*ChatGPTMessage {
	conversation, ok := c.Conversations[chatID]
	if !ok {
		c.ResetConversation(chatID, c.PromptInit)
		conversation = c.Conversations[chatID]
	}

	conversation.Messages = append(conversation.Messages, &ChatGPTMessage{
		Role:    role,
		Content: message,
	})

	if len(conversation.Messages) > 13 {
		len := len(conversation.Messages)
		conversation.Messages = append(conversation.Messages[:1],
			conversation.Messages[len-12:len]...)
	}

	return conversation.Messages
}

func (c *ChatGPT) UpdateUsage(chatID string, usage *ChatGPTUsage) {
	c.Usage.PromptTokens += usage.PromptTokens
	c.Usage.CompletionTokens += usage.CompletionTokens
	c.Usage.TotalTokens += usage.TotalTokens
	if conversation, ok := c.Conversations[chatID]; ok {
		conversation.Usage.PromptTokens += usage.PromptTokens
		conversation.Usage.CompletionTokens += usage.CompletionTokens
		conversation.Usage.TotalTokens += usage.TotalTokens
	}
}

func (c *ChatGPT) SendMessage(chatID string, message string) (*ChatResponse, error) {
	messages := c.UpdateMessage(chatID, "user", message)
	msg, _ := json.Marshal(messages)
	c.Logger.Info(string(msg))
	requestBody, err := json.Marshal(ChatGPTRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    messages,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions",
		bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.OpenAIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"error sending message: %v",
			resp.Status)
	}

	var response ChatGPTResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices")
	}

	reply := response.Choices[0].Message.Content
	c.UpdateMessage(chatID, "assistant", reply)
	c.UpdateUsage(chatID, &response.Usage)
	reply += fmt.Sprintf("\n\n本次花费`%v`个tokens", response.Usage.TotalTokens)
	return &ChatResponse{Message: reply}, nil
}
