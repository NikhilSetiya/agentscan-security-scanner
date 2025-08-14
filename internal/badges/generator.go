package badges

import (
	"fmt"
	"net/url"
	"strings"
)

// BadgeStyle represents different badge styles
type BadgeStyle string

const (
	StyleFlat       BadgeStyle = "flat"
	StyleFlatSquare BadgeStyle = "flat-square"
	StyleForTheBadge BadgeStyle = "for-the-badge"
	StylePlastic    BadgeStyle = "plastic"
	StyleSocial     BadgeStyle = "social"
)

// BadgeColor represents badge colors
type BadgeColor string

const (
	ColorBrightGreen BadgeColor = "brightgreen"
	ColorGreen       BadgeColor = "green"
	ColorYellow      BadgeColor = "yellow"
	ColorYellowGreen BadgeColor = "yellowgreen"
	ColorOrange      BadgeColor = "orange"
	ColorRed         BadgeColor = "red"
	ColorBlue        BadgeColor = "blue"
	ColorLightGrey   BadgeColor = "lightgrey"
)

// SecurityBadge represents a security badge configuration
type SecurityBadge struct {
	Label       string     `json:"label"`
	Message     string     `json:"message"`
	Color       BadgeColor `json:"color"`
	Style       BadgeStyle `json:"style"`
	Logo        string     `json:"logo,omitempty"`
	LogoColor   string     `json:"logo_color,omitempty"`
	Link        string     `json:"link,omitempty"`
	Description string     `json:"description,omitempty"`
}

// BadgeGenerator generates security badges for repositories
type BadgeGenerator struct {
	baseURL string
}

// NewBadgeGenerator creates a new badge generator
func NewBadgeGenerator() *BadgeGenerator {
	return &BadgeGenerator{
		baseURL: "https://img.shields.io/badge",
	}
}

// GenerateSecurityBadge creates a security badge based on scan results
func (bg *BadgeGenerator) GenerateSecurityBadge(scanResults ScanResults) SecurityBadge {
	if scanResults.TotalFindings == 0 {
		return SecurityBadge{
			Label:       "Secured by",
			Message:     "AgentScan",
			Color:       ColorBrightGreen,
			Style:       StyleForTheBadge,
			Logo:        "shield",
			LogoColor:   "white",
			Link:        "https://agentscan.dev?utm_source=badge&utm_medium=github&utm_campaign=security",
			Description: "This repository is secured by AgentScan with no security vulnerabilities detected",
		}
	}

	if scanResults.HighSeverity > 0 {
		return SecurityBadge{
			Label:       "Security",
			Message:     fmt.Sprintf("%d issues found", scanResults.TotalFindings),
			Color:       ColorRed,
			Style:       StyleForTheBadge,
			Logo:        "shield",
			LogoColor:   "white",
			Link:        "https://agentscan.dev?utm_source=badge&utm_medium=github&utm_campaign=security",
			Description: fmt.Sprintf("Security scan found %d issues including %d high severity", scanResults.TotalFindings, scanResults.HighSeverity),
		}
	}

	if scanResults.MediumSeverity > 0 {
		return SecurityBadge{
			Label:       "Security",
			Message:     fmt.Sprintf("%d issues found", scanResults.TotalFindings),
			Color:       ColorYellow,
			Style:       StyleForTheBadge,
			Logo:        "shield",
			LogoColor:   "white",
			Link:        "https://agentscan.dev?utm_source=badge&utm_medium=github&utm_campaign=security",
			Description: fmt.Sprintf("Security scan found %d issues including %d medium severity", scanResults.TotalFindings, scanResults.MediumSeverity),
		}
	}

	return SecurityBadge{
		Label:       "Security",
		Message:     fmt.Sprintf("%d low issues", scanResults.LowSeverity),
		Color:       ColorYellowGreen,
		Style:       StyleForTheBadge,
		Logo:        "shield",
		LogoColor:   "white",
		Link:        "https://agentscan.dev?utm_source=badge&utm_medium=github&utm_campaign=security",
		Description: fmt.Sprintf("Security scan found %d low severity issues", scanResults.LowSeverity),
	}
}

// GenerateBadgeURL creates a shields.io badge URL
func (bg *BadgeGenerator) GenerateBadgeURL(badge SecurityBadge) string {
	// Encode label and message for URL
	label := url.QueryEscape(badge.Label)
	message := url.QueryEscape(badge.Message)
	
	// Build base URL
	badgeURL := fmt.Sprintf("%s/%s-%s-%s", bg.baseURL, label, message, badge.Color)
	
	// Add query parameters
	params := url.Values{}
	
	if badge.Style != "" {
		params.Add("style", string(badge.Style))
	}
	
	if badge.Logo != "" {
		params.Add("logo", badge.Logo)
	}
	
	if badge.LogoColor != "" {
		params.Add("logoColor", badge.LogoColor)
	}
	
	if len(params) > 0 {
		badgeURL += "?" + params.Encode()
	}
	
	return badgeURL
}

// GenerateMarkdownBadge creates a markdown badge with link
func (bg *BadgeGenerator) GenerateMarkdownBadge(badge SecurityBadge) string {
	badgeURL := bg.GenerateBadgeURL(badge)
	
	if badge.Link != "" {
		return fmt.Sprintf("[![%s](%s)](%s)", badge.Description, badgeURL, badge.Link)
	}
	
	return fmt.Sprintf("![%s](%s)", badge.Description, badgeURL)
}

// GenerateHTMLBadge creates an HTML badge with link
func (bg *BadgeGenerator) GenerateHTMLBadge(badge SecurityBadge) string {
	badgeURL := bg.GenerateBadgeURL(badge)
	
	if badge.Link != "" {
		return fmt.Sprintf(`<a href="%s"><img src="%s" alt="%s"></a>`, badge.Link, badgeURL, badge.Description)
	}
	
	return fmt.Sprintf(`<img src="%s" alt="%s">`, badgeURL, badge.Description)
}

// GenerateRESTBadge creates a reStructuredText badge
func (bg *BadgeGenerator) GenerateRESTBadge(badge SecurityBadge) string {
	badgeURL := bg.GenerateBadgeURL(badge)
	
	if badge.Link != "" {
		return fmt.Sprintf(".. image:: %s\n   :target: %s\n   :alt: %s", badgeURL, badge.Link, badge.Description)
	}
	
	return fmt.Sprintf(".. image:: %s\n   :alt: %s", badgeURL, badge.Description)
}

// GenerateAsciiDocBadge creates an AsciiDoc badge
func (bg *BadgeGenerator) GenerateAsciiDocBadge(badge SecurityBadge) string {
	badgeURL := bg.GenerateBadgeURL(badge)
	
	if badge.Link != "" {
		return fmt.Sprintf("image:%s[\"%s\", link=\"%s\"]", badgeURL, badge.Description, badge.Link)
	}
	
	return fmt.Sprintf("image:%s[\"%s\"]", badgeURL, badge.Description)
}

// GetPredefinedBadges returns a set of predefined AgentScan badges
func (bg *BadgeGenerator) GetPredefinedBadges() map[string]SecurityBadge {
	return map[string]SecurityBadge{
		"secured": {
			Label:       "Secured by",
			Message:     "AgentScan",
			Color:       ColorBrightGreen,
			Style:       StyleForTheBadge,
			Logo:        "shield",
			LogoColor:   "white",
			Link:        "https://agentscan.dev?utm_source=badge&utm_medium=github&utm_campaign=predefined",
			Description: "Secured by AgentScan",
		},
		"scanned": {
			Label:       "Security",
			Message:     "Scanned",
			Color:       ColorBlue,
			Style:       StyleForTheBadge,
			Logo:        "shield",
			LogoColor:   "white",
			Link:        "https://agentscan.dev?utm_source=badge&utm_medium=github&utm_campaign=predefined",
			Description: "Security scanned by AgentScan",
		},
		"monitored": {
			Label:       "Security",
			Message:     "Monitored",
			Color:       ColorGreen,
			Style:       StyleForTheBadge,
			Logo:        "shield",
			LogoColor:   "white",
			Link:        "https://agentscan.dev?utm_source=badge&utm_medium=github&utm_campaign=predefined",
			Description: "Security monitored by AgentScan",
		},
		"protected": {
			Label:       "Protected by",
			Message:     "AgentScan",
			Color:       ColorBrightGreen,
			Style:       StyleFlat,
			Logo:        "shield",
			LogoColor:   "white",
			Link:        "https://agentscan.dev?utm_source=badge&utm_medium=github&utm_campaign=predefined",
			Description: "Protected by AgentScan multi-agent security scanning",
		},
	}
}

// GenerateBadgeInstructions creates instructions for adding badges to README
func (bg *BadgeGenerator) GenerateBadgeInstructions(badge SecurityBadge) BadgeInstructions {
	return BadgeInstructions{
		Markdown:  bg.GenerateMarkdownBadge(badge),
		HTML:      bg.GenerateHTMLBadge(badge),
		REST:      bg.GenerateRESTBadge(badge),
		AsciiDoc:  bg.GenerateAsciiDocBadge(badge),
		URL:       bg.GenerateBadgeURL(badge),
		Badge:     badge,
	}
}

// GenerateViralBadges creates badges optimized for viral growth
func (bg *BadgeGenerator) GenerateViralBadges(scanResults ScanResults) []BadgeInstructions {
	var badges []BadgeInstructions
	
	// Main security badge
	mainBadge := bg.GenerateSecurityBadge(scanResults)
	badges = append(badges, bg.GenerateBadgeInstructions(mainBadge))
	
	// Powered by badge
	poweredByBadge := SecurityBadge{
		Label:       "Powered by",
		Message:     "AgentScan",
		Color:       ColorBlue,
		Style:       StyleFlat,
		Logo:        "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEyIDJMMTMuMDkgOC4yNkwyMCA5TDEzLjA5IDE1Ljc0TDEyIDIyTDEwLjkxIDE1Ljc0TDQgOUwxMC45MSA4LjI2TDEyIDJaIiBmaWxsPSJ3aGl0ZSIvPgo8L3N2Zz4K",
		Link:        "https://agentscan.dev?utm_source=badge&utm_medium=github&utm_campaign=viral",
		Description: "Powered by AgentScan multi-agent security scanning",
	}
	badges = append(badges, bg.GenerateBadgeInstructions(poweredByBadge))
	
	// Multi-agent consensus badge
	consensusBadge := SecurityBadge{
		Label:       "Multi-Agent",
		Message:     "Consensus",
		Color:       ColorBrightGreen,
		Style:       StyleFlat,
		Logo:        "checkmark",
		LogoColor:   "white",
		Link:        "https://agentscan.dev/features/consensus?utm_source=badge&utm_medium=github&utm_campaign=viral",
		Description: "Multi-agent consensus security scanning",
	}
	badges = append(badges, bg.GenerateBadgeInstructions(consensusBadge))
	
	return badges
}

// ScanResults represents the results of a security scan
type ScanResults struct {
	TotalFindings  int `json:"total_findings"`
	HighSeverity   int `json:"high_severity"`
	MediumSeverity int `json:"medium_severity"`
	LowSeverity    int `json:"low_severity"`
	ScanID         string `json:"scan_id"`
	Repository     string `json:"repository"`
}

// BadgeInstructions contains badge code in different formats
type BadgeInstructions struct {
	Markdown string        `json:"markdown"`
	HTML     string        `json:"html"`
	REST     string        `json:"rest"`
	AsciiDoc string        `json:"asciidoc"`
	URL      string        `json:"url"`
	Badge    SecurityBadge `json:"badge"`
}

// GenerateREADMESection creates a complete README section with badges
func (bg *BadgeGenerator) GenerateREADMESection(scanResults ScanResults) string {
	badges := bg.GenerateViralBadges(scanResults)
	
	var section strings.Builder
	
	section.WriteString("## 🛡️ Security\n\n")
	
	// Add badges
	for _, badge := range badges {
		section.WriteString(badge.Markdown + " ")
	}
	section.WriteString("\n\n")
	
	// Add security summary
	if scanResults.TotalFindings == 0 {
		section.WriteString("This repository has been scanned by [AgentScan](https://agentscan.dev) and no security vulnerabilities were detected.\n\n")
		section.WriteString("**Security Features:**\n")
		section.WriteString("- ✅ Multi-agent consensus scanning\n")
		section.WriteString("- ✅ SAST, SCA, and secrets detection\n")
		section.WriteString("- ✅ Continuous security monitoring\n")
		section.WriteString("- ✅ Zero known vulnerabilities\n\n")
	} else {
		section.WriteString(fmt.Sprintf("This repository is monitored by [AgentScan](https://agentscan.dev). Last scan detected %d security findings.\n\n", scanResults.TotalFindings))
		
		if scanResults.HighSeverity > 0 {
			section.WriteString(fmt.Sprintf("⚠️ **%d high severity** issues require immediate attention.\n", scanResults.HighSeverity))
		}
		if scanResults.MediumSeverity > 0 {
			section.WriteString(fmt.Sprintf("🔶 **%d medium severity** issues should be addressed.\n", scanResults.MediumSeverity))
		}
		if scanResults.LowSeverity > 0 {
			section.WriteString(fmt.Sprintf("🔵 **%d low severity** issues for consideration.\n", scanResults.LowSeverity))
		}
		section.WriteString("\n")
	}
	
	section.WriteString("**Want to add security scanning to your project?**\n")
	section.WriteString("1. Sign up at [agentscan.dev](https://agentscan.dev/signup?utm_source=readme&utm_medium=github&utm_campaign=viral)\n")
	section.WriteString("2. Add the [GitHub Action](https://github.com/marketplace/actions/agentscan-security-scanner)\n")
	section.WriteString("3. Install the [VS Code extension](https://marketplace.visualstudio.com/items?itemName=agentscan.agentscan-security)\n\n")
	
	return section.String()
}

// GenerateBadgeAPI creates an API response for badge generation
func (bg *BadgeGenerator) GenerateBadgeAPI(scanResults ScanResults, format string) interface{} {
	switch strings.ToLower(format) {
	case "shields":
		badge := bg.GenerateSecurityBadge(scanResults)
		return map[string]interface{}{
			"schemaVersion": 1,
			"label":         badge.Label,
			"message":       badge.Message,
			"color":         string(badge.Color),
			"style":         string(badge.Style),
			"logoSvg":       badge.Logo,
		}
	case "json":
		return bg.GenerateViralBadges(scanResults)
	default:
		badge := bg.GenerateSecurityBadge(scanResults)
		return bg.GenerateBadgeInstructions(badge)
	}
}