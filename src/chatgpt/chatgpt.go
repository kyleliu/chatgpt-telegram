package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const CHATGPT_BASELINE = "你是一个代码写作助手，你的名字叫多多。"

type ChatGPTConversation struct {
	ChatID   int64
	Messages []*ChatGPTMessage
	Usage    *ChatGPTUsage
}

type ChatGPT struct {
	OpenAIKey     string
	Conversations map[int64]*ChatGPTConversation
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

func Init(apiKey string) *ChatGPT {
	return &ChatGPT{
		OpenAIKey:     apiKey,
		Conversations: map[int64]*ChatGPTConversation{},
	}
}

func (c *ChatGPT) ResetConversation(chatID int64, basecmd string) {
	messages := []*ChatGPTMessage{}
	if basecmd != "" {
		messages = append(messages, &ChatGPTMessage{
			Role:    "system",
			Content: basecmd,
		})
	}
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

func (c *ChatGPT) UpdateMessage(chatID int64, role string, message string) []*ChatGPTMessage {
	conversation, ok := c.Conversations[chatID]
	if !ok {
		c.ResetConversation(chatID, CHATGPT_BASELINE)
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

func (c *ChatGPT) UpdateUsage(chatID int64, usage *ChatGPTUsage) {
	if conversation, ok := c.Conversations[chatID]; ok {
		conversation.Usage.PromptTokens += usage.PromptTokens
		conversation.Usage.CompletionTokens += usage.CompletionTokens
		conversation.Usage.TotalTokens += usage.TotalTokens
	}
}

func (c *ChatGPT) SendMessage(message string, chatID int64) (*ChatResponse, error) {
	messages := c.UpdateMessage(chatID, "user", message)
	requestBody, err := json.Marshal(ChatGPTRequest{
		Model:       "gpt-3.5-turbo",
		Messages:    messages,
		Temperature: 0.1,
	})
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions",
		bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.OpenAIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error sending message: %v", err)
		return nil, fmt.Errorf(
			"error sending message: %v",
			resp.Status)
	}

	var response ChatGPTResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return nil, err
	}

	if len(response.Choices) == 0 {
		log.Printf("Error sending message: %v", err)
		return nil, fmt.Errorf("no choices")
	}

	reply := response.Choices[0].Message.Content
	c.UpdateMessage(chatID, "assistant", reply)
	c.UpdateUsage(chatID, &response.Usage)
	reply += fmt.Sprintf("\n\n本次花费`%v`个tokens", response.Usage.TotalTokens)
	return &ChatResponse{Message: reply}, nil
}
