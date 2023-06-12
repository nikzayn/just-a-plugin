package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"

	openai "github.com/openai/openai-go/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	chat "google.golang.org/api/chat/v1"
	"google.golang.org/api/option"
	"gopkg.in/joho/godotenv.v1"
)

var (
	questionPattern = regexp.MustCompile(`^\?[\s\u200c]*`)
)

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func createGoogleChatService() (*chat.SpacesMessagesService, error) {
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, []byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")), chat.ChatBotScope)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Chat service: %v", err)
	}

	client := oauth2.NewClient(ctx, creds.TokenSource)
	service, err := chat.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Chat service: %v", err)
	}

	return service.Spaces.Messages, nil
}

func processQuestion(question string) (string, error) {
	// Initialize the OpenAI API client
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	// Define the conversation history
	conversation := []openai.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: question},
	}

	// Call the OpenAI ChatGPT API
	response, err := client.CompleteConversation(context.Background(), os.Getenv("OPENAI_MODEL_NAME"), conversation, nil)
	if err != nil {
		return "", fmt.Errorf("failed to process question: %v", err)
	}

	// Extract the generated response
	generatedResponse := response.Choices[0].Message.Content
	return generatedResponse, nil
}

func handleChatEvent(event *chat.Event) {
	message := event.Message

	if message != nil && questionPattern.MatchString(message.Text) {
		question := questionPattern.ReplaceAllString(message.Text, "")
		response, err := processQuestion(question)
		if err != nil {
			log.Printf("Error processing question: %v", err)
			return
		}

		// TODO: Send the response back to the Google Meet chat using the Google Chat API

		log.Printf("Question: %s", question)
		log.Printf("Response: %s", response)
		log.Println("-----")
	}
}

func main() {
	loadEnv()

	// Create Google Chat service
	googleChatService, err := createGoogleChatService()
	if err != nil {
		log.Fatalf("Failed to create Google Chat service: %v", err)
	}

	// Start listening to chat events
	events := make(chan *chat.Event)
	err = googleChatService.Watch(os.Getenv("GOOGLE_CHAT_SPACE_NAME"), &chat.WatchRequest{
		Filter: "text",
	}).Do(func(watchResponse *chat.WatchResponse) {
		log.Printf("Connected to Google Chat: %s", watchResponse.Space.Name)
		go func() {
			for _, e := range watchResponse.Events {
				events <- e
			}
		}()
	})
	if err != nil {
		log.Fatalf("Failed to watch Google Chat space: %v", err)
	}

	// Handle chat events
	for event := range events {
		handleChatEvent(event)
	}
}
