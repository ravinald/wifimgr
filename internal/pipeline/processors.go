package pipeline

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// GrepProcessor implements grep functionality
type GrepProcessor struct{}

// Name returns the processor name
func (g *GrepProcessor) Name() string {
	return "grep"
}

// Usage returns usage information
func (g *GrepProcessor) Usage() string {
	return "grep <pattern> - filter lines matching pattern (supports regex)"
}

// Process filters lines matching the pattern
func (g *GrepProcessor) Process(input io.Reader, args []string) (io.Reader, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("grep requires a pattern argument")
	}

	pattern := args[0]
	invertMatch := false

	// Check for -v flag (invert match)
	if len(args) > 1 && args[0] == "-v" {
		if len(args) < 2 {
			return nil, fmt.Errorf("grep -v requires a pattern argument")
		}
		invertMatch = true
		pattern = args[1]
	}

	// Compile regex pattern
	regex, err := regexp.Compile(pattern)
	if err != nil {
		// If regex compilation fails, treat as literal string
		return g.processLiteral(input, pattern, invertMatch)
	}

	return g.processRegex(input, regex, invertMatch)
}

// processRegex processes input using regex matching
func (g *GrepProcessor) processRegex(input io.Reader, regex *regexp.Regexp, invertMatch bool) (io.Reader, error) {
	var matchedLines []string
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()
		matches := regex.MatchString(line)

		if (matches && !invertMatch) || (!matches && invertMatch) {
			matchedLines = append(matchedLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return LinesToReader(matchedLines), nil
}

// processLiteral processes input using literal string matching
func (g *GrepProcessor) processLiteral(input io.Reader, pattern string, invertMatch bool) (io.Reader, error) {
	var matchedLines []string
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()
		contains := strings.Contains(line, pattern)

		if (contains && !invertMatch) || (!contains && invertMatch) {
			matchedLines = append(matchedLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return LinesToReader(matchedLines), nil
}

// HeadProcessor implements head functionality
type HeadProcessor struct{}

// Name returns the processor name
func (h *HeadProcessor) Name() string {
	return "head"
}

// Usage returns usage information
func (h *HeadProcessor) Usage() string {
	return "head [-n lines] - show first N lines (default: 10)"
}

// Process returns the first N lines
func (h *HeadProcessor) Process(input io.Reader, args []string) (io.Reader, error) {
	numLines := 10 // default

	// Parse arguments
	if len(args) > 0 {
		if len(args) >= 2 && args[0] == "-n" {
			var err error
			numLines, err = strconv.Atoi(args[1])
			if err != nil {
				return nil, fmt.Errorf("invalid number of lines: %s", args[1])
			}
		} else if len(args) == 1 {
			// Try to parse as number directly
			if num, err := strconv.Atoi(args[0]); err == nil {
				numLines = num
			} else {
				return nil, fmt.Errorf("invalid argument: %s", args[0])
			}
		}
	}

	if numLines < 0 {
		return nil, fmt.Errorf("number of lines cannot be negative")
	}

	var lines []string
	scanner := bufio.NewScanner(input)
	count := 0

	for scanner.Scan() && count < numLines {
		lines = append(lines, scanner.Text())
		count++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return LinesToReader(lines), nil
}

// TailProcessor implements tail functionality
type TailProcessor struct{}

// Name returns the processor name
func (t *TailProcessor) Name() string {
	return "tail"
}

// Usage returns usage information
func (t *TailProcessor) Usage() string {
	return "tail [-n lines] - show last N lines (default: 10)"
}

// Process returns the last N lines
func (t *TailProcessor) Process(input io.Reader, args []string) (io.Reader, error) {
	numLines := 10 // default

	// Parse arguments
	if len(args) > 0 {
		if len(args) >= 2 && args[0] == "-n" {
			var err error
			numLines, err = strconv.Atoi(args[1])
			if err != nil {
				return nil, fmt.Errorf("invalid number of lines: %s", args[1])
			}
		} else if len(args) == 1 {
			// Try to parse as number directly
			if num, err := strconv.Atoi(args[0]); err == nil {
				numLines = num
			} else {
				return nil, fmt.Errorf("invalid argument: %s", args[0])
			}
		}
	}

	if numLines < 0 {
		return nil, fmt.Errorf("number of lines cannot be negative")
	}

	// Read all lines first
	var allLines []string
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Get the last N lines
	startIndex := len(allLines) - numLines
	if startIndex < 0 {
		startIndex = 0
	}

	tailLines := allLines[startIndex:]
	return LinesToReader(tailLines), nil
}

// CreateDefaultRegistry creates a registry with default processors
func CreateDefaultRegistry() *ProcessorRegistry {
	registry := NewProcessorRegistry()

	// Register default processors
	registry.Register(&GrepProcessor{})
	registry.Register(&HeadProcessor{})
	registry.Register(&TailProcessor{})

	return registry
}
