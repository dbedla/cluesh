package cluesh

import (
	"os"

	"github.com/dbedla/rellm/pkg/rellm"
	"github.com/invopop/jsonschema"
	"github.com/joho/godotenv"
)

// The struct tags describe the schema.
type Person struct {
	Name string `json:"name" jsonschema:"description=Full name of the person"`
	Age  int    `json:"age"  jsonschema:"description=Age in years"`
	City string `json:"city" jsonschema:"description=City of residence"`
}

func NewAgent() *rellm.Agent {

	textFormat := rellm.TextFormat{
		Type:   "json_schema",
		Name:   "person",
		Strict: true,
		Schema: (&jsonschema.Reflector{DoNotReference: true}).Reflect(&Person{}),
	}

	_ = godotenv.Load() // reads OPENROUTER_API_KEY from .env
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		panic("OPENROUTER_API_KEY is not set")
	}

	provider, err := rellm.NewOpenRouterProvider(apiKey, "openai/gpt-5.6-luna")
	if err != nil {
		panic(err)
	}

	agent, err := rellm.NewAgentBuilder().
		WithProvider(provider).
		WithAgentName("StructuredOutputAgent").
		WithMaxAgentSteps(20).
		WithConversation(rellm.NewInMemoryConversation()).
		WithImageGenerationKeepInTheLoop().
		WithUnknownConversationElementKeepInTheLoop().
		WithSystemMessage("You are an assistant that extracts structured information from user text and returns only valid JSON matching the requested schema.").
		WithTextFormat(textFormat).
		Build()
	if err != nil {
		panic(err)
	}

	return agent
}
