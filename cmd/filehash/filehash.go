package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/ravinald/wifimgr/internal/filehash"
	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/symbols"
)

// findJsonFiles recursively finds all .json files in the given directory
// and adds their absolute paths to the targetFiles slice
func findJsonFiles(directory string, targetFiles *[]string) {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing %s: %v\n", path, err)
			return nil // continue walking despite error
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include .json files
		if strings.HasSuffix(strings.ToLower(info.Name()), ".json") {
			// Avoid .meta files
			if !strings.HasSuffix(path, ".meta") {
				*targetFiles = append(*targetFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory %s: %v\n", directory, err)
	}
}

func main() {
	// Initialize logging
	if err := logging.ConfigureLogger("info", false); err != nil {
		fmt.Printf("Failed to configure logger: %v\n", err)
	}

	// Parse command line flags
	verify := flag.Bool("verify", false, "Verify file integrity")
	generate := flag.Bool("generate", false, "Generate metadata for file")
	description := flag.String("desc", "wifimgr file", "Description of the file")
	allFiles := flag.Bool("all", false, "Process all .json files in ./cache and ./config directories")
	singleFile := flag.String("f", "", "Process a single file (can be used with --generate or --verify)")
	flag.Parse()

	// Get the file path from arguments
	var targetFiles []string

	if *allFiles {
		// Find all .json files in ./cache and ./config directories
		configDir, err := filepath.Abs("./config")
		if err == nil {
			findJsonFiles(configDir, &targetFiles)
		} else {
			fmt.Printf("Error resolving config directory: %v\n", err)
		}

		cacheDir, err := filepath.Abs("./cache")
		if err == nil {
			findJsonFiles(cacheDir, &targetFiles)
		} else {
			fmt.Printf("Error resolving cache directory: %v\n", err)
		}

		if len(targetFiles) == 0 {
			fmt.Println("No .json files found in ./cache or ./config directories")
		}
	} else if *singleFile != "" {
		// Process a single file specified with -f flag
		absPath, err := filepath.Abs(*singleFile)
		if err != nil {
			fmt.Printf("Error resolving path for %s: %v\n", *singleFile, err)
			os.Exit(1)
		}

		// Check if file exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			fmt.Printf("File %s does not exist\n", absPath)
			os.Exit(1)
		}

		targetFiles = append(targetFiles, absPath)
	} else if flag.NArg() > 0 {
		// Use specified file(s) as positional arguments
		for _, file := range flag.Args() {
			absPath, err := filepath.Abs(file)
			if err != nil {
				fmt.Printf("Error resolving path for %s: %v\n", file, err)
				continue
			}

			// Check if file exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				fmt.Printf("File %s does not exist, skipping\n", absPath)
				continue
			}

			targetFiles = append(targetFiles, absPath)
		}
	} else {
		// No file specified, -f not provided, and --all not used
		fmt.Println("Please specify a file path, use -f flag, or use --all flag")
		flag.Usage()
		os.Exit(1)
	}

	// Process each file
	for _, filePath := range targetFiles {
		if *generate {
			// Generate metadata
			fmt.Printf("Generating metadata for %s...\n", filePath)
			if err := filehash.CreateMetadataFile(filePath, *description); err != nil {
				fmt.Printf("Error generating metadata: %v\n", err)
				continue
			}
			fmt.Printf("%s Metadata generated successfully for %s\n",
				symbols.SuccessPrefix(), filePath)
		}

		if *verify {
			// Verify file integrity
			fmt.Printf("Verifying integrity of %s...\n", filePath)
			status, err := filehash.VerifyFile(filePath)
			if err != nil {
				fmt.Printf("Error verifying file: %v\n", err)
				continue
			}

			switch status {
			case filehash.FileOK:
				fmt.Printf("%s File integrity verified for %s\n",
					symbols.SuccessPrefix(), filePath)
			case filehash.FileNew:
				fmt.Printf("%s New metadata created for %s\n",
					color.YellowString("!"), filePath)
			case filehash.FileRegenerated:
				fmt.Printf("%s Metadata regenerated for %s\n",
					color.YellowString("!"), filePath)
			case filehash.FileFailed:
				fmt.Printf("%s File integrity check failed for %s\n",
					symbols.FailurePrefix(), filePath)
			}
		}

		// If neither verify nor generate was specified, display file hash
		if !*verify && !*generate {
			hash, err := filehash.CalculateFileHash(filePath)
			if err != nil {
				fmt.Printf("Error calculating hash: %v\n", err)
				continue
			}
			fmt.Printf("File: %s\nHash: %s\n\n", filePath, hash)
		}
	}
}
