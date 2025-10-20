package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sebrandon1/ocp-doc-checker/pkg/checker"
)

// URLLocation tracks where a URL appears in the codebase
type URLLocation struct {
	URL   string
	Files []string // Files where this URL appears
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// Flags
	urlFlag          = flag.String("url", "", "OCP documentation URL to check")
	dirFlag          = flag.String("dir", "", "Directory or file to scan for OCP documentation URLs")
	fixFlag          = flag.Bool("fix", false, "Automatically fix outdated URLs in files (only works with -dir)")
	verboseFlag      = flag.Bool("verbose", false, "Enable verbose output")
	jsonFlag         = flag.Bool("json", false, "Output results in JSON format")
	versionFlag      = flag.Bool("version", false, "Print version information")
	allAvailableFlag = flag.Bool("all-available", false, "Show all available newer versions (default: latest only)")
)

func main() {
	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("ocp-doc-checker %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Validate flags - ensure mutual exclusivity
	if *urlFlag == "" && *dirFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: either -url or -dir flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if *urlFlag != "" && *dirFlag != "" {
		fmt.Fprintln(os.Stderr, "Error: -url and -dir flags are mutually exclusive")
		flag.Usage()
		os.Exit(1)
	}

	if *fixFlag && *dirFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: -fix flag can only be used with -dir flag")
		flag.Usage()
		os.Exit(1)
	}

	if *fixFlag && *jsonFlag {
		fmt.Fprintln(os.Stderr, "Error: -fix flag cannot be used with -json flag")
		flag.Usage()
		os.Exit(1)
	}

	// Create checker
	c := checker.NewChecker()

	// Handle based on mode
	if *urlFlag != "" {
		// Single URL mode
		handleSingleURL(c, *urlFlag)
	} else {
		// Directory/file scan mode
		handleDirectory(c, *dirFlag)
	}
}

func handleSingleURL(c *checker.Checker, url string) {
	// Perform check
	result, err := c.Check(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking URL: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if *jsonFlag {
		printJSONResults(result)
	} else {
		printTextResults(result, *verboseFlag)
	}

	// Exit with appropriate code
	if result.IsOutdated {
		os.Exit(1) // Exit with error code if documentation is outdated
	}
}

func handleDirectory(c *checker.Checker, path string) {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing path: %v\n", err)
		os.Exit(1)
	}

	// Collect URLs with their locations
	var urlLocations []URLLocation
	if info.IsDir() {
		urlLocations, err = scanDirectoryWithLocations(path)
	} else {
		urlLocations, err = scanFileWithLocations(path)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning for URLs: %v\n", err)
		os.Exit(1)
	}

	if len(urlLocations) == 0 {
		if !*jsonFlag {
			fmt.Println("✅ No OCP Documentation URLs found")
		}
		os.Exit(0)
	}

	if !*jsonFlag {
		fmt.Printf("Found %d unique OCP documentation URL(s)\n\n", len(urlLocations))
	}

	// Check all URLs
	var results []*checker.CheckResult
	hasOutdated := false
	urlToLocation := make(map[string]URLLocation)

	for i, loc := range urlLocations {
		if *verboseFlag {
			fmt.Printf("[%d/%d] Checking: %s\n", i+1, len(urlLocations), loc.URL)
		}

		urlToLocation[loc.URL] = loc

		result, err := c.Check(loc.URL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking URL %s: %v\n", loc.URL, err)
			continue
		}

		results = append(results, result)
		if result.IsOutdated {
			hasOutdated = true
		}
	}

	// Apply fixes if requested
	if *fixFlag && hasOutdated {
		applyFixes(results, urlToLocation)
	}

	// Output results
	if *jsonFlag {
		printBatchJSONResults(results)
	} else {
		printBatchTextResults(results, *verboseFlag)
	}

	// Exit with appropriate code
	if hasOutdated && !*fixFlag {
		os.Exit(1)
	}
}

// scanDirectoryWithLocations recursively scans a directory and tracks URL locations
func scanDirectoryWithLocations(dir string) ([]URLLocation, error) {
	urlToFiles := make(map[string][]string)

	// Supported file extensions
	supportedExts := map[string]bool{
		".md":       true,
		".markdown": true,
		".txt":      true,
		".adoc":     true,
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file has supported extension
		ext := filepath.Ext(path)
		if !supportedExts[ext] {
			return nil
		}

		// Scan the file
		fileURLs, err := scanFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error scanning %s: %v\n", path, err)
			return nil // Continue with other files
		}

		// Track which files contain which URLs
		for _, url := range fileURLs {
			urlToFiles[url] = append(urlToFiles[url], path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice of URLLocation
	var locations []URLLocation
	for url, files := range urlToFiles {
		// Deduplicate files
		fileSet := make(map[string]bool)
		for _, f := range files {
			fileSet[f] = true
		}
		uniqueFiles := make([]string, 0, len(fileSet))
		for f := range fileSet {
			uniqueFiles = append(uniqueFiles, f)
		}

		locations = append(locations, URLLocation{
			URL:   url,
			Files: uniqueFiles,
		})
	}

	return locations, nil
}

// scanFileWithLocations scans a single file and returns URL locations
func scanFileWithLocations(path string) ([]URLLocation, error) {
	urls, err := scanFile(path)
	if err != nil {
		return nil, err
	}

	// Deduplicate URLs for this single file
	urlSet := make(map[string]bool)
	for _, url := range urls {
		urlSet[url] = true
	}

	var locations []URLLocation
	for url := range urlSet {
		locations = append(locations, URLLocation{
			URL:   url,
			Files: []string{path},
		})
	}

	return locations, nil
}

// scanFile scans a single file for OCP documentation URLs
func scanFile(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	urlRegex := regexp.MustCompile(`https://docs\.redhat\.com/[^\s)\]"]*openshift_container_platform/\d+\.\d+/[^\s)\]"]*`)
	matches := urlRegex.FindAllString(string(content), -1)

	// Clean up URLs (remove trailing punctuation)
	var cleanedURLs []string
	for _, url := range matches {
		cleaned := strings.TrimRight(url, ".,;:!?")
		cleanedURLs = append(cleanedURLs, cleaned)
	}

	return cleanedURLs, nil
}

// applyFixes updates files with the latest URLs
func applyFixes(results []*checker.CheckResult, urlToLocation map[string]URLLocation) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("🔧 Applying Fixes...")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	fixedFiles := make(map[string]bool)
	fixCount := 0

	for _, result := range results {
		if !result.IsOutdated || len(result.NewerVersions) == 0 {
			continue
		}

		// Get the latest version URL
		latest := result.NewerVersions[len(result.NewerVersions)-1]
		oldURL := result.OriginalURL
		newURL := latest.URL

		// Get the files containing this URL
		location, ok := urlToLocation[oldURL]
		if !ok {
			continue
		}

		// Update each file
		for _, filePath := range location.Files {
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filePath, err)
				continue
			}

			// Replace all occurrences of the old URL with the new URL
			newContent := strings.ReplaceAll(string(content), oldURL, newURL)

			if newContent != string(content) {
				err = os.WriteFile(filePath, []byte(newContent), 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filePath, err)
					continue
				}

				fixedFiles[filePath] = true
				fixCount++

				fmt.Printf("✅ Updated: %s\n", filePath)
				fmt.Printf("   %s → %s\n", result.OriginalVersion, latest.Version)
				fmt.Printf("   Old: %s\n", oldURL)
				fmt.Printf("   New: %s\n\n", newURL)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Summary: Fixed %d URL(s) in %d file(s)\n", fixCount, len(fixedFiles))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
}

func printTextResults(result *checker.CheckResult, verbose bool) {
	fmt.Printf("Checking: %s\n", result.OriginalURL)
	fmt.Printf("Current Version: %s\n", result.OriginalVersion)
	fmt.Println(strings.Repeat("-", 80))

	if result.IsOutdated {
		fmt.Printf("⚠️  This documentation is OUTDATED!\n")
		fmt.Printf("Latest Version: %s\n\n", result.LatestVersion)

		if *allAvailableFlag {
			fmt.Println("Available newer versions:")
			for _, v := range result.NewerVersions {
				fmt.Printf("  ✓ Version %s: %s\n", v.Version, v.URL)
			}
		} else {
			// Show only the latest version
			if len(result.NewerVersions) > 0 {
				latest := result.NewerVersions[len(result.NewerVersions)-1]
				fmt.Printf("Latest available version:\n")
				fmt.Printf("  ✓ Version %s: %s\n", latest.Version, latest.URL)

				if len(result.NewerVersions) > 1 {
					fmt.Printf("\n(Use --all-available to see all %d newer versions)\n", len(result.NewerVersions))
				}
			}
		}

		if verbose {
			fmt.Println("\nAll checked versions:")
			for _, v := range result.AllResults {
				status := "✗ Not found"
				if v.Exists {
					if v.HasAnchor {
						if v.AnchorExists {
							status = "✓ Found (page + anchor)"
						} else {
							status = "⚠ Page found, anchor missing"
						}
					} else {
						status = "✓ Found"
					}
				}
				fmt.Printf("  %s Version %s: %s\n", status, v.Version, v.URL)
			}
		}
	} else {
		fmt.Printf("✓ This documentation is UP TO DATE (version %s)\n", result.LatestVersion)

		// Check if there are newer versions with missing anchors
		missingAnchors := []checker.VersionCheckResult{}
		for _, v := range result.AllResults {
			if v.Exists && v.HasAnchor && !v.AnchorExists {
				missingAnchors = append(missingAnchors, v)
			}
		}

		if len(missingAnchors) > 0 {
			fmt.Println("\n⚠️  Note: Newer versions exist but the anchor is missing:")
			for _, v := range missingAnchors {
				fmt.Printf("  - Version %s: page exists but anchor not found\n", v.Version)
			}
		}

		if verbose {
			fmt.Println("\nChecked versions:")
			for _, v := range result.AllResults {
				status := "✗ Not found"
				if v.Exists {
					if v.HasAnchor {
						if v.AnchorExists {
							status = "✓ Found (page + anchor)"
						} else {
							status = "⚠ Page found, anchor missing"
						}
					} else {
						status = "✓ Found"
					}
				}
				fmt.Printf("  %s Version %s\n", status, v.Version)
			}
		}
	}
}

func printJSONResults(result *checker.CheckResult) {
	fmt.Printf(`{
  "original_url": "%s",
  "original_version": "%s",
  "latest_version": "%s",
  "is_outdated": %t,
  "newer_versions": [
`, result.OriginalURL, result.OriginalVersion, result.LatestVersion, result.IsOutdated)

	for i, v := range result.NewerVersions {
		comma := ","
		if i == len(result.NewerVersions)-1 {
			comma = ""
		}
		fmt.Printf(`    {"version": "%s", "url": "%s"}%s
`, v.Version, v.URL, comma)
	}

	fmt.Println(`  ]`)
	fmt.Println(`}`)
}

func printBatchTextResults(results []*checker.CheckResult, verbose bool) {
	uptodateCount := 0
	outdatedCount := 0

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("📋 OCP Documentation URL Check Results")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	for i, result := range results {
		if result.IsOutdated {
			outdatedCount++
			fmt.Printf("[%d] ⚠️  OUTDATED\n", i+1)
		} else {
			uptodateCount++
			fmt.Printf("[%d] ✅ UP TO DATE\n", i+1)
		}

		fmt.Printf("    URL: %s\n", result.OriginalURL)
		fmt.Printf("    Current Version: %s\n", result.OriginalVersion)
		fmt.Printf("    Latest Version: %s\n", result.LatestVersion)

		if result.IsOutdated && len(result.NewerVersions) > 0 {
			if *allAvailableFlag {
				fmt.Println("    Available newer versions:")
				for _, v := range result.NewerVersions {
					fmt.Printf("      - Version %s: %s\n", v.Version, v.URL)
				}
			} else {
				latest := result.NewerVersions[len(result.NewerVersions)-1]
				fmt.Printf("    Latest available: %s (%s)\n", latest.Version, latest.URL)
				if len(result.NewerVersions) > 1 {
					fmt.Printf("    (%d newer versions available, use --all-available to see all)\n", len(result.NewerVersions))
				}
			}
		}

		fmt.Println()
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Summary: %d total, %d up-to-date, %d outdated\n", len(results), uptodateCount, outdatedCount)
	fmt.Println(strings.Repeat("=", 80))

	// Print recommendations for outdated URLs
	if outdatedCount > 0 {
		fmt.Println()
		fmt.Println("🔧 Recommended Updates:")
		fmt.Println()
		for _, result := range results {
			if result.IsOutdated && len(result.NewerVersions) > 0 {
				latest := result.NewerVersions[len(result.NewerVersions)-1]
				fmt.Printf("- Update from %s to %s:\n", result.OriginalVersion, latest.Version)
				fmt.Printf("  Old: %s\n", result.OriginalURL)
				fmt.Printf("  New: %s\n", latest.URL)
				fmt.Println()
			}
		}
	}
}

func printBatchJSONResults(results []*checker.CheckResult) {
	uptodateCount := 0
	outdatedCount := 0

	for _, result := range results {
		if result.IsOutdated {
			outdatedCount++
		} else {
			uptodateCount++
		}
	}

	fmt.Printf(`{
  "total_count": %d,
  "uptodate_count": %d,
  "outdated_count": %d,
  "results": [
`, len(results), uptodateCount, outdatedCount)

	for i, result := range results {
		comma := ","
		if i == len(results)-1 {
			comma = ""
		}

		fmt.Printf(`    {
      "original_url": "%s",
      "original_version": "%s",
      "latest_version": "%s",
      "is_outdated": %t,
      "newer_versions": [`, result.OriginalURL, result.OriginalVersion, result.LatestVersion, result.IsOutdated)

		for j, v := range result.NewerVersions {
			versionComma := ","
			if j == len(result.NewerVersions)-1 {
				versionComma = ""
			}
			fmt.Printf(`
        {"version": "%s", "url": "%s"}%s`, v.Version, v.URL, versionComma)
		}

		fmt.Printf(`
      ]
    }%s
`, comma)
	}

	fmt.Println(`  ]`)
	fmt.Println(`}`)
}
