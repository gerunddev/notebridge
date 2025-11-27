package main

import (
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "org-to-md", "o2m":
		handleOrgToMarkdown(os.Args[2:])
	case "md-to-org", "m2o":
		handleMarkdownToOrg(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("notebridge v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	usage := `notebridge - Convert notes between org-mode and markdown

Usage:
  notebridge <command> [options]

Commands:
  org-to-md, o2m    Convert org-mode/org-roam files to markdown
  md-to-org, m2o    Convert markdown files to org-mode
  version           Show version information
  help              Show this help message

Examples:
  notebridge org-to-md notes/file.org
  notebridge md-to-org notes/file.md
  notebridge org-to-md --dir notes/org --out notes/md

For more information, visit: https://github.com/gerunddev/notebridge
`
	fmt.Print(usage)
}

func handleOrgToMarkdown(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No input file specified")
		os.Exit(1)
	}

	inputFile := args[0]
	fmt.Printf("Converting org-mode to markdown: %s\n", inputFile)
	fmt.Println("(conversion not yet implemented)")
	// TODO: Implement org-to-markdown conversion
}

func handleMarkdownToOrg(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No input file specified")
		os.Exit(1)
	}

	inputFile := args[0]
	fmt.Printf("Converting markdown to org-mode: %s\n", inputFile)
	fmt.Println("(conversion not yet implemented)")
	// TODO: Implement markdown-to-org conversion
}
