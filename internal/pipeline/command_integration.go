package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/logging"
)

// CommandOutput captures the output from a command execution
type CommandOutput struct {
	Content string
	Error   error
}

// CommandHandler is a function that executes a command and returns output
type CommandHandler func(ctx context.Context, client api.Client, cfg *config.Config, args []string, formatOverride string, force bool) (string, error)

// PipelineManager manages pipeline execution for commands
type PipelineManager struct {
	registry *ProcessorRegistry
	executor *PipelineExecutor
}

// NewPipelineManager creates a new pipeline manager
func NewPipelineManager() *PipelineManager {
	registry := CreateDefaultRegistry()
	executor := NewPipelineExecutor(registry)

	return &PipelineManager{
		registry: registry,
		executor: executor,
	}
}

// ExecuteWithPipeline executes a command with optional pipeline processing
func (pm *PipelineManager) ExecuteWithPipeline(ctx context.Context, client api.Client, cfg *config.Config, args []string, formatOverride string, force bool, handler CommandHandler) error {
	// Check if this command has a pipeline
	if !HasPipeline(args) {
		// No pipeline, execute command normally and print output
		output, err := handler(ctx, client, cfg, args, formatOverride, force)
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	}

	// Parse the pipeline
	pipeline, err := ParsePipeline(args)
	if err != nil {
		return fmt.Errorf("failed to parse pipeline: %w", err)
	}

	logging.Debugf("Executing pipeline: base=%v, pipes=%v", pipeline.BaseCommand, pipeline.PipeCommands)

	// Execute base command and capture output
	output, err := handler(ctx, client, cfg, pipeline.BaseCommand, formatOverride, force)
	if err != nil {
		return err
	}

	// Process through pipeline
	input := StringToReader(output)
	result, err := pm.executor.Execute(input, pipeline)
	if err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	// Print final result
	resultStr, err := ReaderToString(result)
	if err != nil {
		return fmt.Errorf("failed to read pipeline result: %w", err)
	}

	fmt.Print(resultStr)
	return nil
}

// CaptureOutput captures stdout during function execution
func CaptureOutput(fn func() error) (string, error) {
	// Create a pipe to capture output
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	// Replace stdout with write end of pipe
	os.Stdout = w

	// Channel to capture the output
	outputChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Start goroutine to read from pipe
	go func() {
		defer close(outputChan)
		defer close(errorChan)

		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			errorChan <- err
			return
		}
		outputChan <- buf.String()
	}()

	// Execute function
	fnErr := fn()

	// Close write end and restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Wait for output
	select {
	case output := <-outputChan:
		if fnErr != nil {
			return output, fnErr
		}
		return output, nil
	case err := <-errorChan:
		if fnErr != nil {
			return "", fnErr
		}
		return "", err
	}
}

// WrapCommandHandler wraps a command handler to return output instead of printing
func WrapCommandHandler(originalHandler func(ctx context.Context, client api.Client, cfg *config.Config, args []string, formatOverride string, force bool) error) CommandHandler {
	return func(ctx context.Context, client api.Client, cfg *config.Config, args []string, formatOverride string, force bool) (string, error) {
		output, err := CaptureOutput(func() error {
			return originalHandler(ctx, client, cfg, args, formatOverride, force)
		})
		return output, err
	}
}

// GetRegistry returns the processor registry
func (pm *PipelineManager) GetRegistry() *ProcessorRegistry {
	return pm.registry
}

// GetAvailableProcessors returns a list of available processors
func (pm *PipelineManager) GetAvailableProcessors() []string {
	processors := pm.registry.GetAll()
	var names []string
	for name := range processors {
		names = append(names, name)
	}
	return names
}

// GetProcessorUsage returns usage information for a processor
func (pm *PipelineManager) GetProcessorUsage(name string) string {
	if processor, exists := pm.registry.Get(name); exists {
		return processor.Usage()
	}
	return fmt.Sprintf("Unknown processor: %s", name)
}
