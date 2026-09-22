package cluesh

import (
	"fmt"
	"os"

	"github.com/dbedla/rellm/pkg/rellm"
	"github.com/joho/godotenv"
)

func BuildOpenRouterProvider(model rellm.Model) (rellm.Provider, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("cannot load .env: %w", err)
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing apikey for OPENROUTER_API_KEY")
	}

	return rellm.NewOpenRouterProvider(apiKey, model)
}

func BuildOpenAIProvider(model rellm.Model) (rellm.Provider, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("cannot load .env: %w", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing apikey for OPENAI_API_KEY")
	}

	return rellm.NewOpenAIProvider(apiKey, model)
}

func BuildLMSProvider(model rellm.Model) (rellm.Provider, error) {
	return rellm.NewLMStudioProvider(model, "http://127.0.0.1", "1234")
}
