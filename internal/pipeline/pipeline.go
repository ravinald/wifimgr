package pipeline

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// PipeCommand represents a single command in a pipeline
type PipeCommand struct {
	Name string
	Args []string
}

// Pipeline represents a complete command pipeline
type Pipeline struct {
	BaseCommand  []string
	PipeCommands []PipeCommand
}

// Processor interface for pipeline command processors
type Processor interface {
	// Name returns the name of the processor (e.g., "grep", "head", "tail")
	Name() string

	// Process takes input and arguments, returns processed output
	Process(input io.Reader, args []string) (io.Reader, error)

	// Usage returns usage information for the processor
	Usage() string
}

// ProcessorRegistry manages available pipeline processors
type ProcessorRegistry struct {
	processors map[string]Processor
}

// NewProcessorRegistry creates a new processor registry
func NewProcessorRegistry() *ProcessorRegistry {
	return &ProcessorRegistry{
		processors: make(map[string]Processor),
	}
}

// Register adds a processor to the registry
func (r *ProcessorRegistry) Register(processor Processor) {
	r.processors[processor.Name()] = processor
}

// Get retrieves a processor by name
func (r *ProcessorRegistry) Get(name string) (Processor, bool) {
	processor, exists := r.processors[name]
	return processor, exists
}

// GetAll returns all registered processors
func (r *ProcessorRegistry) GetAll() map[string]Processor {
	result := make(map[string]Processor)
	for name, processor := range r.processors {
		result[name] = processor
	}
	return result
}

// PipelineExecutor executes a pipeline of commands
type PipelineExecutor struct {
	registry *ProcessorRegistry
}

// NewPipelineExecutor creates a new pipeline executor
func NewPipelineExecutor(registry *ProcessorRegistry) *PipelineExecutor {
	return &PipelineExecutor{
		registry: registry,
	}
}

// Execute runs the pipeline on the given input
func (e *PipelineExecutor) Execute(input io.Reader, pipeline Pipeline) (io.Reader, error) {
	current := input

	for i, pipeCmd := range pipeline.PipeCommands {
		processor, exists := e.registry.Get(pipeCmd.Name)
		if !exists {
			return nil, fmt.Errorf("unknown pipe command: %s", pipeCmd.Name)
		}

		processed, err := processor.Process(current, pipeCmd.Args)
		if err != nil {
			return nil, fmt.Errorf("error in pipe command %d (%s): %w", i+1, pipeCmd.Name, err)
		}

		current = processed
	}

	return current, nil
}

// ParsePipeline parses command arguments into a pipeline structure
func ParsePipeline(args []string) (Pipeline, error) {
	if len(args) == 0 {
		return Pipeline{}, fmt.Errorf("empty command")
	}

	pipeline := Pipeline{}

	// Find pipe symbols and split the command
	var currentSegment []string
	var segments [][]string
	lastWasPipe := false

	for _, arg := range args {
		if arg == "|" {
			if len(currentSegment) == 0 {
				return Pipeline{}, fmt.Errorf("empty command segment before pipe")
			}
			segments = append(segments, currentSegment)
			currentSegment = nil
			lastWasPipe = true
		} else {
			currentSegment = append(currentSegment, arg)
			lastWasPipe = false
		}
	}

	// Check if command ended with a pipe
	if lastWasPipe {
		return Pipeline{}, fmt.Errorf("command cannot end with pipe")
	}

	// Add the last segment
	if len(currentSegment) > 0 {
		segments = append(segments, currentSegment)
	}

	if len(segments) == 0 {
		return Pipeline{}, fmt.Errorf("no command segments found")
	}

	// First segment is the base command
	pipeline.BaseCommand = segments[0]

	// Remaining segments are pipe commands
	for i := 1; i < len(segments); i++ {
		segment := segments[i]
		if len(segment) == 0 {
			return Pipeline{}, fmt.Errorf("empty pipe command at position %d", i)
		}

		pipeCmd := PipeCommand{
			Name: segment[0],
			Args: segment[1:],
		}
		pipeline.PipeCommands = append(pipeline.PipeCommands, pipeCmd)
	}

	return pipeline, nil
}

// HasPipeline returns true if the arguments contain pipe commands
func HasPipeline(args []string) bool {
	for _, arg := range args {
		if arg == "|" {
			return true
		}
	}
	return false
}

// StringToReader converts a string to an io.Reader
func StringToReader(s string) io.Reader {
	return strings.NewReader(s)
}

// ReaderToString converts an io.Reader to a string
func ReaderToString(r io.Reader) (string, error) {
	var result strings.Builder
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		result.WriteString(scanner.Text())
		result.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return result.String(), nil
}

// ReaderToLines converts an io.Reader to a slice of lines
func ReaderToLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// LinesToReader converts a slice of lines to an io.Reader
func LinesToReader(lines []string) io.Reader {
	return strings.NewReader(strings.Join(lines, "\n"))
}
