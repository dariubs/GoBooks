// Validator for README.md structure in github.com/dariubs/gobooks.
//
// Checks: required sections, TOC link validity, no year prefixes in titles,
// book metadata (*Last published*, *Authors*), cover images exist, covers linked.
//
// Usage: go run . [path/to/README.md]
//        go run .              # uses README.md in parent dir (../README.md)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	readmePath := filepath.Join("..", "README.md")
	if len(os.Args) > 1 {
		readmePath = os.Args[1]
	}

	content, err := os.ReadFile(readmePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	lines := strings.Split(string(content), "\n")
	errors := []string{}
	warnings := []string{}

	// 1. Check main sections exist
	requiredSections := []string{"Starter Books", "Advanced Books", "Web Development", "Resources"}
	foundSections := extractSections(lines)
	for _, sec := range requiredSections {
		if !foundSections[sec] {
			errors = append(errors, fmt.Sprintf("Missing required section: ## %s", sec))
		}
	}
	// Contributing and License may use ## or ==== format
	hasContributing := foundSections["Contributing"]
	hasLicense := foundSections["License"]
	for i, line := range lines {
		if strings.TrimSpace(line) == "Contributing" && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "====") {
			hasContributing = true
		}
		if strings.TrimSpace(line) == "License" && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "====") {
			hasLicense = true
		}
	}
	if !hasContributing {
		errors = append(errors, "Missing required section: Contributing")
	}
	if !hasLicense {
		errors = append(errors, "Missing required section: License")
	}

	// 2. Extract TOC links and section headers
	tocLinks := extractTOCLinks(lines)
	headers := extractHeaders(lines)
	// Add Contributing/License anchors when using ==== format
	if hasContributing {
		headers["contributing"] = true
	}
	if hasLicense {
		headers["license"] = true
	}

	// 3. Verify TOC links point to existing headers
	for link, anchor := range tocLinks {
		if !headers[anchor] {
			errors = append(errors, fmt.Sprintf("TOC link points to non-existent anchor: [%s](#%s)", truncate(link, 50), anchor))
		}
	}

	// 4. Check section headers have no year prefix (### 2020 - [Title] is wrong)
	yearPrefix := regexp.MustCompile(`^###\s+\d{4}\s+-\s+`)
	for i, line := range lines {
		if yearPrefix.MatchString(line) {
			errors = append(errors, fmt.Sprintf("Line %d: Book title should not have year prefix: %s", i+1, strings.TrimSpace(line)[:min(60, len(strings.TrimSpace(line)))]+"..."))
		}
	}

	// 5. Check book entries have required metadata (*Last published* and *Authors*) - only in book sections
	bookBlocks := extractBookBlocks(lines)
	inResourceSection := false
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Resources" {
			inResourceSection = true
			break
		}
		_ = i
	}
	resourceTitles := map[string]bool{"A tour of Go": true, "Video: Learn Go Syntax in one video": true, "Tutorials: Go by Example": true, "Go Fundamentals Video Training": true, "More Books on the Go Wiki": true, "TutorialEdge.net Course": true, "Coursera Specialization: Programming with Go": true, "Course: Understand Go's In-Depth Mechanics": true, "Course: Mastering Go Programming": true, "Course: Web Development with Google's Go Programming Language": true, "Golangbot.com Articles": true, "Tuxerrante repo on go exercises": true}
	for title, block := range bookBlocks {
		if resourceTitles[title] || inResourceSection && (strings.Contains(title, "tour") || strings.Contains(title, "Video") || strings.Contains(title, "Tutorial") || strings.Contains(title, "Course") || strings.Contains(title, "Articles") || strings.Contains(title, "Wiki")) {
			continue
		}
		hasLastPublished := strings.Contains(block, "*Last published*")
		hasAuthors := strings.Contains(block, "*Authors:*")
		if !hasLastPublished {
			warnings = append(warnings, fmt.Sprintf("Book '%s' missing *Last published*", truncate(title, 50)))
		}
		if !hasAuthors {
			warnings = append(warnings, fmt.Sprintf("Book '%s' missing *Authors:*", truncate(title, 50)))
		}
	}

	// 6. Check cover images exist
	coverRefs := extractCoverRefs(lines)
	for _, ref := range coverRefs {
		if strings.HasPrefix(ref, "http") {
			continue // external URL, skip
		}
		path := filepath.Join(filepath.Dir(readmePath), ref)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("Cover image not found: %s", ref))
		}
	}

	// 7. Check cover images are linked (wrapped in <a href>)
	unlinkedCovers := findUnlinkedCovers(lines)
	for _, img := range unlinkedCovers {
		warnings = append(warnings, fmt.Sprintf("Cover image not linked to book: %s", img))
	}

	// Report
	if len(errors) > 0 {
		fmt.Println("❌ ERRORS:")
		for _, e := range errors {
			fmt.Println("  ", e)
		}
	}
	if len(warnings) > 0 {
		fmt.Println("\n⚠️  WARNINGS:")
		for _, w := range warnings {
			fmt.Println("  ", w)
		}
	}

	if len(errors) > 0 {
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Println("\n✓ Structure OK (with warnings)")
	} else {
		fmt.Println("\n✓ Structure OK")
	}
}

func extractSections(lines []string) map[string]bool {
	m := make(map[string]bool)
	r := regexp.MustCompile(`^##\s+(.+)$`)
	for _, line := range lines {
		if match := r.FindStringSubmatch(line); match != nil {
			m[strings.TrimSpace(match[1])] = true
		}
	}
	return m
}

func extractHeaders(lines []string) map[string]bool {
	m := make(map[string]bool)
	r := regexp.MustCompile(`^#{1,6}\s+(.+)$`)
	for _, line := range lines {
		if match := r.FindStringSubmatch(line); match != nil {
			text := strings.TrimSpace(match[1])
			// Extract link text from [Title](url) or [Title](url) *Free*
			linkMatch := regexp.MustCompile(`^\[([^\]]+)\][^\)]*\)`).FindStringSubmatch(text)
			if linkMatch != nil {
				text = linkMatch[1]
			}
			// Remove *Free* suffix
			text = regexp.MustCompile(`\s*\*Free\*\s*$`).ReplaceAllString(text, "")
			text = strings.TrimSpace(text)
			anchor := toAnchor(text)
			m[anchor] = true
			// Add -free variant (TOC often uses this for free books even when header omits *Free*)
			m[anchor+"-free"] = true
		}
	}
	return m
}

func toAnchor(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^a-z0-9\s-]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func extractTOCLinks(lines []string) map[string]string {
	m := make(map[string]string)
	r := regexp.MustCompile(`^\s*-\s+\[([^\]]+)\]\(#([^\)]+)\)`)
	for _, line := range lines {
		if match := r.FindStringSubmatch(line); match != nil {
			title := strings.TrimSpace(match[1])
			anchor := strings.TrimSpace(match[2])
			m[title] = anchor
		}
	}
	return m
}

func extractBookBlocks(lines []string) map[string]string {
	m := make(map[string]string)
	bookR := regexp.MustCompile(`^###\s+\[([^\]]+)\].*$`)
	var currentTitle string
	var currentBlock strings.Builder

	for _, line := range lines {
		if match := bookR.FindStringSubmatch(line); match != nil {
			if currentTitle != "" {
				m[currentTitle] = currentBlock.String()
			}
			currentTitle = match[1]
			currentBlock.Reset()
			currentBlock.WriteString(line)
			currentBlock.WriteString("\n")
		} else if currentTitle != "" && (strings.HasPrefix(line, "###") || strings.HasPrefix(line, "##")) {
			m[currentTitle] = currentBlock.String()
			currentTitle = ""
			currentBlock.Reset()
		} else if currentTitle != "" && line != "-----" {
			currentBlock.WriteString(line)
			currentBlock.WriteString("\n")
		}
	}
	if currentTitle != "" {
		m[currentTitle] = currentBlock.String()
	}
	return m
}

func extractCoverRefs(lines []string) []string {
	var refs []string
	r := regexp.MustCompile(`src="(covers/[^"]+)"`)
	for _, line := range lines {
		if match := r.FindStringSubmatch(line); match != nil {
			refs = append(refs, match[1])
		}
	}
	// Also check cover/ path
	r2 := regexp.MustCompile(`src="(cover/[^"]+)"`)
	for _, line := range lines {
		if match := r2.FindStringSubmatch(line); match != nil {
			refs = append(refs, match[1])
		}
	}
	return refs
}

func findUnlinkedCovers(lines []string) []string {
	var unlinked []string
	r := regexp.MustCompile(`<img\s+src="(covers/[^"]+)"`)
	for _, line := range lines {
		// Skip if img is inside <a href> or markdown link [<img...>](url)
		if (strings.Contains(line, "<a href=") || strings.Contains(line, "](http")) && strings.Contains(line, "<img") {
			continue
		}
		if match := r.FindStringSubmatch(line); match != nil {
			unlinked = append(unlinked, match[1])
		}
	}
	return unlinked
}

func truncate(s string, max int) string {
	s = strings.Split(s, "(")[0]
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
