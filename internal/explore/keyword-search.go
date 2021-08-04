package explore

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bm402/gander/internal/logger"
)

type grepResult struct {
	filename      string
	line          string
	matchedString string
}

type collectedResult struct {
	filename string
	line     string
	count    int
}

func SearchLogsForVariableAssignments(owner, repo, wordlistPath string) {
	variableNames := getWordsFromWordlist(wordlistPath)
	logger.Print(owner, repo, "search-variables", "Read", len(variableNames), "variable names from wordlist")

	assigners := []string{"=", ":"}
	for _, variableName := range variableNames {
		for _, assigner := range assigners {
			collectedResults := make(map[string]collectedResult)
			grepResults := searchCurrentDirectoryUsingGrep(owner, repo, variableName+assigner)

			for _, result := range grepResults {
				if _, ok := collectedResults[result.matchedString]; ok {
					updatedCollectedResult := collectedResults[result.matchedString]
					updatedCollectedResult.count++
					collectedResults[result.matchedString] = updatedCollectedResult
				} else {
					collectedResults[result.matchedString] = collectedResult{
						filename: result.filename,
						line:     result.line,
						count:    1,
					}
				}
			}

			for matchedString, detail := range collectedResults {
				logger.Print(owner, repo, "search-variables", "Found", matchedString, "at",
					detail.filename+":"+detail.line+",", "and", detail.count-1, "other occurrences")
			}
		}
	}
}

func SearchLogsForMissingDependencies(owner, repo, wordlistPath string) {
	missingDependencyMessages := getWordsFromWordlist(wordlistPath)
	logger.Print(owner, repo, "search-dependencies", "Read", len(missingDependencyMessages), "missing dependency messages from wordlist")

	for _, missingDependencyMessage := range missingDependencyMessages {
		collectedResults := make(map[string]collectedResult)
		grepResults := searchCurrentDirectoryUsingGrep(owner, repo, missingDependencyMessage)

		for _, result := range grepResults {
			if _, ok := collectedResults[result.matchedString]; ok {
				updatedCollectedResult := collectedResults[result.matchedString]
				updatedCollectedResult.count++
				collectedResults[result.matchedString] = updatedCollectedResult
			} else {
				collectedResults[result.matchedString] = collectedResult{
					filename: result.filename,
					line:     result.line,
					count:    1,
				}
			}
		}

		for matchedString, detail := range collectedResults {
			logger.Print(owner, repo, "search-dependencies", "Found", matchedString, "at",
				detail.filename+":"+detail.line+",", "and", detail.count-1, "other occurrences")
		}
	}
}

func getWordsFromWordlist(wordlistPath string) []string {
	file, err := os.Open(wordlistPath)
	if err != nil {
		logger.Print("Could not read wordlist from", wordlistPath+":", err.Error())
		return []string{}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	words := []string{}
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}

	return words
}

func searchCurrentDirectoryUsingGrep(owner, repo, str string) []grepResult {
	grepResults := []grepResult{}
	grepOutput, err := exec.Command("grep", "-nri", str, ".").Output()
	if err != nil && err.Error() != "exit status 1" {
		logger.Print(owner, repo, "search-grep", "Could not search for", str, "using grep:", err.Error())
	}

	grepOutputLines := strings.Split(string(grepOutput), "\n")
	for _, grepOutputLine := range grepOutputLines {

		parts := strings.Split(grepOutputLine, ":")
		if len(parts) < 3 {
			continue
		}
		partsCount := 0
		filename := parts[partsCount]
		partsCount++

		var line string
		for {
			if _, err := strconv.Atoi(parts[partsCount]); err != nil {
				filename += ":" + parts[partsCount]
			} else {
				line = parts[partsCount]
				break
			}
			partsCount++
		}

		matchedString := strings.Join(parts[partsCount:], ":")
		matchedStringParts := strings.Fields(matchedString)
		sanitisedMatchedString := strings.Join(matchedStringParts[1:], " ")

		grepResults = append(grepResults, grepResult{
			filename:      filename,
			line:          line,
			matchedString: sanitisedMatchedString,
		})
	}

	return grepResults
}
