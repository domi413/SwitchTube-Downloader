// Package auth provides functions for managing the user's access token
package auth

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	configDirName = "/switchtube-dl"
	tokenFileName = "/token.txt"

	createAccessToken = "https://tube.switch.ch/access_tokens"
)

func getTokenDir() string {
	dirname, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	return dirname + configDirName
}

func GetToken() string {
	token, err := os.ReadFile(getTokenDir() + tokenFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		} else {
			log.Fatal(err)
		}
	}

	return strings.TrimSpace(string(token))
}

func CreateToken() ([]byte, error) {
	fmt.Printf("Go to: %s and paste the token here: ", createAccessToken)

	reader := bufio.NewReader(os.Stdin)
	token, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read token: %w", err)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}

	fmt.Print("Do you want to save the token? (y/n): ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read answer: %w", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "y" || answer == "yes" {
		err = saveToken(token, getTokenDir()+tokenFileName)
		if err != nil {
			return nil, err
		}
	}

	return []byte(token), nil
}

func saveToken(token string, tokenFile string) error {
	if err := os.MkdirAll(getTokenDir(), 0o755); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
	}

	if err := os.WriteFile(tokenFile, []byte(token), 0o600); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	fmt.Printf("Token saved to %s\n", tokenFile)

	return nil
}
