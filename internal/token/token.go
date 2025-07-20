// Package token provides functionality for managing access tokens to
// authenticate with SwitchTube
package token

import (
	"errors"
	"fmt"
	"os/user"

	"switch-tube-downloader/internal/prompt"

	"github.com/zalando/go-keyring"
)

const (
	serviceName          = "SwitchTube"
	createAccessTokenURL = "https://tube.switch.ch/access_tokens"
)

var (
	// ErrTokenAlreadyExists is returned when trying to set a token that already
	// exists in the keyring.
	ErrTokenAlreadyExists = errors.New("token already exists in keyring")

	errTokenEmpty         = errors.New("token cannot be empty")
	errNoTokenFoundDelete = errors.New("no token found in keyring")
	errFailedToGetUser    = errors.New("failed to get current user")
	errUnableToCreate     = errors.New("unable to create access token")
	errFailedToStore      = errors.New("failed to store token in keyring")
	errFailedToDelete     = errors.New("failed to delete token from keyring")
	errFailedToRetrieve   = errors.New("failed to retrieve token from keyring")
	errNoTokenFound       = errors.New("no token found in keyring - run 'token set' first")
)

// Get retrieves the access token from the system keyring.
func Get() (string, error) {
	userName, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("%w: %w", errFailedToGetUser, err)
	}

	token, err := keyring.Get(serviceName, userName.Username)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", errNoTokenFound
		}

		return "", fmt.Errorf("%w: %w", errFailedToRetrieve, err)
	}

	return token, nil
}

// Set creates and stores a new access token in the system keyring.
func Set() error {
	// Check if token already exists
	existingToken, err := Get()
	if err == nil && existingToken != "" {
		fmt.Println("Token already exists in keyring")

		if !prompt.Confirm("Do you want to replace it?") {
			fmt.Println("Operation cancelled")

			return fmt.Errorf("%w", ErrTokenAlreadyExists)
		}
	}

	token, err := create()
	if err != nil {
		return fmt.Errorf("%w: %w", errUnableToCreate, err)
	}

	userName, err := user.Current()
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToGetUser, err)
	}

	err = keyring.Set(serviceName, userName.Username, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToStore, err)
	}

	return nil
}

// Delete removes the access token from the system keyring.
func Delete() error {
	userName, err := user.Current()
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToGetUser, err)
	}

	err = keyring.Delete(serviceName, userName.Username)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return fmt.Errorf("%w for %s", errNoTokenFoundDelete, serviceName)
		}

		return fmt.Errorf("%w: %w", errFailedToDelete, err)
	}

	return nil
}

// create prompts the user to visit the access token creation URL and enter a new token.
func create() (string, error) {
	fmt.Printf("Please visit: %s\n", createAccessTokenURL)
	fmt.Printf("Create a new access token and paste it below\n\n")

	token := prompt.Input("Enter your access token: ")
	if token == "" {
		return "", errTokenEmpty
	}

	return token, nil
}
