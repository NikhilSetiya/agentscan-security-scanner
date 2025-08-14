package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// SARIFReport represents a SARIF 2.1.0 report
type SARIFReport struct {
	Version string      `json:"version"`
	Schema  string      `json:"$schema"`
	Runs    []SARIFRun  `json:"runs"`
}

type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

type SARIFDriver struct {
	Name            string      `json:"name"`
	Version         string      `json:"version"`
	InformationURI  string      `json:"informationUri"`
	Rules           []SARIFRule `json:"rules"`
}

type SARIFRule struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	ShortDescription SARIFMessage           `json:"shortDescription"`
	FullDescription  SARIFMessage           `json:"fullDescription"`
	Help             SARIFMessage           `json:"help"`
	Properties       map[string]interface{} `json:"properties"`
}

type SARIFResult struct {
	RuleID    string         `json:"ruleId"`
	RuleIndex int            `json:"ruleIndex"`
	Level     string         `json:"level"`
	Message   SARIFMessage   `json:"message"`
	Locations []SARIFLocation `json:"locations"`
}

type SARIFMessage struct {
	Text string `json:"text"`
}

type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Region           SARIFRegion           `json:"region"`
}

type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

type SARIFRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
	EndLine     int `json:"endLine,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}

func main() {
	var (
		command    = flag.String("command", "", "Command to run (convert-sarif)")
		inputFile  = flag.String("input", "", "Input file path")
		outputFile = flag.String("output", "", "Output file path")
		repository = flag.String("repository", "", "Repository URL")
		commit     = flag.String("commit", "", "Commit SHA")
	)
	flag.Parse()

	switch *command {
	case "convert-sarif":
		if err := convertToSARIF(*inputFile, *outputFile, *repository, *commit); err != nil {
			log.Fatalf("Failed to convert to SARIF: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Usage: %s -command=convert-sarif -input=<file> -output=<file> -repository=<url> -commit=<sha>\n", os.Args[0])
		os.Exit(1)
	}
}

func convertToSARIF(inputFile, outputFile, repository, commit string) error {
	// Read input file
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Parse AgentScan results
	var results struct {
		Findings []types.Finding `json:"findings"`
	}
	if err := json.Unmarshal(data, &results); err != nil {
		return fmt.Errorf("failed to parse input JSON: %w", err)
	}

	// Convert to SARIF
	sarif := convertFindingsToSARIF(results.Findings, repository)

	// Write SARIF file
	sarifData, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF: %w", err)
	}

	if err := os.WriteFile(outputFile, sarifData, 0644); err != nil {
		return fmt.Errorf("failed to write SARIF file: %w", err)
	}

	return nil
}

func convertFindingsToSARIF(findings []types.Finding, repository string) SARIFReport {
	// Create rule map
	ruleMap := make(map[string]SARIFRule)
	ruleIndex := make(map[string]int)
	var rules []SARIFRule
	var results []SARIFResult

	for _, finding := range findings {
		// Create rule if not exists
		if _, exists := ruleMap[finding.RuleID]; !exists {
			rule := SARIFRule{
				ID:   finding.RuleID,
				Name: finding.Title,
				ShortDescription: SARIFMessage{
					Text: finding.Title,
				},
				FullDescription: SARIFMessage{
					Text: finding.Description,
				},
				Help: SARIFMessage{
					Text: fmt.Sprintf("%s\n\nTool: %s\nCategory: %s", finding.Description, finding.Tool, finding.Category),
				},
				Properties: map[string]interface{}{
					"tool":       finding.Tool,
					"category":   finding.Category,
					"precision":  "high",
					"tags":       []string{"security", finding.Severity},
				},
			}

			ruleMap[finding.RuleID] = rule
			ruleIndex[finding.RuleID] = len(rules)
			rules = append(rules, rule)
		}

		// Convert severity to SARIF level
		level := "note"
		switch finding.Severity {
		case "high":
			level = "error"
		case "medium":
			level = "warning"
		case "low":
			level = "note"
		}

		// Create result
		result := SARIFResult{
			RuleID:    finding.RuleID,
			RuleIndex: ruleIndex[finding.RuleID],
			Level:     level,
			Message: SARIFMessage{
				Text: fmt.Sprintf("%s: %s", finding.Title, finding.Description),
			},
			Locations: []SARIFLocation{
				{
					PhysicalLocation: SARIFPhysicalLocation{
						ArtifactLocation: SARIFArtifactLocation{
							URI: normalizeFilePath(finding.FilePath),
						},
						Region: SARIFRegion{
							StartLine:   finding.LineNumber,
							StartColumn: finding.ColumnNumber,
						},
					},
				},
			},
		}

		results = append(results, result)
	}

	return SARIFReport{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:           "AgentScan",
						Version:        "1.0.0",
						InformationURI: "https://agentscan.dev",
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}
}

func normalizeFilePath(path string) string {
	// Remove leading slash and convert to relative path
	path = strings.TrimPrefix(path, "/")
	
	// Convert to forward slashes
	path = filepath.ToSlash(path)
	
	// Remove current directory prefix
	path = strings.TrimPrefix(path, "./")
	
	return path
}