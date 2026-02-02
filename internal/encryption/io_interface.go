package encryption

import (
	"fmt"
)

// TokenManagerIO defines the interface for input/output operations in the token manager
type TokenManagerIO interface {
	// Print outputs a message without a newline
	Print(msg string)

	// Println outputs a message with a newline
	Println(msg string)

	// Scanln reads a line of input
	Scanln(value *string) error

	// PromptPassword prompts for a password (should be hidden during input)
	PromptPassword(prompt string) (string, error)

	// PromptWithConfirm prompts for a password with confirmation
	PromptWithConfirm(initialPrompt, confirmPrompt string) (string, error)
}

// DefaultIO implements TokenManagerIO using standard methods
type DefaultIO struct{}

func NewDefaultIO() *DefaultIO {
	return &DefaultIO{}
}

// Print outputs a message without a newline
func (io *DefaultIO) Print(msg string) {
	fmt.Print(msg)
}

// Println outputs a message with a newline
func (io *DefaultIO) Println(msg string) {
	fmt.Println(msg)
}

// Scanln reads a line of input
func (io *DefaultIO) Scanln(value *string) error {
	_, err := fmt.Scanln(value)
	return err
}

// PromptPassword prompts for a password using the existing PromptForPassword function
func (io *DefaultIO) PromptPassword(prompt string) (string, error) {
	return PromptForPassword(prompt)
}

// PromptWithConfirm prompts for a password with confirmation
func (io *DefaultIO) PromptWithConfirm(initialPrompt, confirmPrompt string) (string, error) {
	return promptWithConfirmation(initialPrompt, confirmPrompt)
}
