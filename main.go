package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	outputFile  string
	useHTMX     bool
	useAlpine   bool
	showVersion bool
)

const version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:   "plainkit-converter [input]",
	Short: "Convert HTML to Plain Go code",
	Long: `Plain Converter transforms HTML files into Go code using the Plain HTML library.

It supports standard HTML, htmx attributes, and Alpine.js directives.

Examples:
  # Convert HTML from stdin
  echo '<div class="container">Hello</div>' | plainkit-converter

  # Convert HTML file
  plainkit-converter index.html

  # Convert with htmx support
  plainkit-converter --htmx index.html

  # Convert with Alpine.js support
  plainkit-converter --alpine index.html

  # Save to file
  plainkit-converter index.html -o component.go

  # Convert with both htmx and Alpine.js
  plainkit-converter --htmx --alpine index.html`,

	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Printf("Plain Converter v%s\n", version)
			return nil
		}

		var input io.Reader
		var inputName string

		// Determine input source
		if len(args) > 0 {
			// Read from file
			inputName = args[0]
			file, err := os.Open(inputName)
			if err != nil {
				return fmt.Errorf("failed to open input file: %w", err)
			}
			defer func() {
				if err := file.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "Error closing file: %v\n", err)
				}
			}()
			input = file
		} else {
			// Read from stdin
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				// No stdin input
				return fmt.Errorf("no input provided. Use a file argument or pipe HTML to stdin")
			}
			input = os.Stdin
			inputName = "stdin"
		}

		// Read input
		htmlContent, err := io.ReadAll(input)
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Convert HTML to Plain
		converter := NewConverter(useHTMX, useAlpine)
		goCode, err := converter.Convert(string(htmlContent))
		if err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}

		// Determine output
		if outputFile != "" {
			// Write to file
			err = os.WriteFile(outputFile, []byte(goCode), 0644)
			if err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			fmt.Printf("✓ Converted %s → %s\n", inputName, outputFile)
		} else {
			// Write to stdout
			fmt.Print(goCode)
		}

		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	rootCmd.Flags().BoolVar(&useHTMX, "htmx", false, "Enable htmx attribute conversion")
	rootCmd.Flags().BoolVar(&useAlpine, "alpine", false, "Enable Alpine.js attribute conversion")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
