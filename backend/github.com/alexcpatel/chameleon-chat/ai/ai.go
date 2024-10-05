package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/alexcpatel/chameleon-chat/history"
)

// Character struct definition
type Character struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	VoicePrompt string `json:"voicePrompt"`
}

// CharacterData struct to hold all character-related data
type CharacterData struct {
	SystemPrompt string      `json:"systemPrompt"`
	Characters   []Character `json:"characters"`
}

var characterData CharacterData

func init() {
	// Read the JSON file
	fileContent, err := os.ReadFile("characters.json")
	if err != nil {
		log.Fatalf("Error reading characters.json file: %v", err)
	}

	// Unmarshal the JSON data into the characterData variable
	err = json.Unmarshal(fileContent, &characterData)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON data: %v", err)
	}
}

func GetCharacters() []Character {
	return append([]Character{}, characterData.Characters...)
}

func getCharacterByName(name string) (Character, bool) {
	for _, char := range characterData.Characters {
		if char.Name == name {
			return char, true
		}
	}
	return Character{}, false
}

func createPrompt(characterName string, userMessage string) (string, error) {
	character, found := getCharacterByName(characterName)
	if !found {
		return "", fmt.Errorf("character not found: %s", characterName)
	}

	prompt := characterData.SystemPrompt + "\n\n"
	prompt += "===== CHARACTER INFORMATION =====\n"
	prompt += "Character Name: " + character.Name + "\n"
	prompt += "Description: " + character.Description + "\n"
	prompt += "Voice Prompt: " + character.VoicePrompt + "\n\n"

	prompt += "===== CONVERSATION HISTORY =====\n"
	messages := history.GetHistory(5)
	for _, msg := range messages {
		prompt += "User ID: " + string(msg.ClientID) + "\n"
		prompt += "User: " + msg.RawMsg + "\n"
		prompt += "AI: " + msg.AiMsg + "\n\n"
	}

	prompt += "===== TRANSLATION TASK =====\n"
	prompt += "Original message: !!! '" + userMessage + "' !!!\n\n"
	prompt += "Translated message: "

	return prompt, nil
}

func GenerateMessage(characterName string, userMessage string) (string, error) {
	prompt, err := createPrompt(characterName, userMessage)
	if err != nil {
		return "", err
	}

	response, err := callClaudeAPI(prompt)
	if err != nil {
		return "", fmt.Errorf("error calling Claude API: %v", err)
	}

	return response, nil
}
