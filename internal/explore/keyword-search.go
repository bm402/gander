package explore

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

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

func SearchLogsForVariableAssignments(owner, repo, wordlistPath string, threads int) int {
	variableNames := getWordsFromWordlist(wordlistPath)
	logger.Print(owner, repo, "search-variables", "Read", len(variableNames), "variable names from wordlist")

	wg := sync.WaitGroup{}
	variableAssignments := make(chan string, 2*len(variableNames))
	distinctMatchesFound := int64(0)

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(owner, repo string, variableAssignments <-chan string) {
			for variableAssignment := range variableAssignments {
				collectedResults := collectSearchResults(owner, repo, variableAssignment)
				for matchedString, detail := range collectedResults {
					atomic.AddInt64(&distinctMatchesFound, 1)
					logger.Print(owner, repo, "search-variables", "Found", matchedString, "at",
						detail.filename+":"+detail.line+",", "and", detail.count-1, "other occurrences")
				}
				wg.Done()
			}
		}(owner, repo, variableAssignments)
	}

	// add variable assignments to channel to trigger workers
	assigners := []string{"=", ":"}
	for _, variableName := range variableNames {
		for _, assigner := range assigners {
			wg.Add(1)
			variableAssignments <- variableName + assigner
		}
	}

	// close channel and wait for threads to finish
	close(variableAssignments)
	wg.Wait()

	return int(distinctMatchesFound)
}

func SearchLogsForKeywords(owner, repo, wordlistPath string, threads int) int {
	keywords := getWordsFromWordlist(wordlistPath)
	logger.Print(owner, repo, "search-keywords", "Read", len(keywords), "keywords from wordlist")

	wg := sync.WaitGroup{}
	keywordsChan := make(chan string, len(keywords))
	distinctMatchesFound := int64(0)

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(owner, repo string, keywordsChan <-chan string) {
			for keyword := range keywordsChan {
				collectedResults := collectSearchResults(owner, repo, keyword)
				for matchedString, detail := range collectedResults {
					atomic.AddInt64(&distinctMatchesFound, 1)
					logger.Print(owner, repo, "search-keywords", "Found", matchedString, "at",
						detail.filename+":"+detail.line+",", "and", detail.count-1, "other occurrences")
				}
				wg.Done()
			}
		}(owner, repo, keywordsChan)
	}

	// add keywords to channel to trigger workers
	for _, keyword := range keywords {
		wg.Add(1)
		keywordsChan <- keyword
	}

	// close channel and wait for threads to finish
	close(keywordsChan)
	wg.Wait()

	return int(distinctMatchesFound)
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

func collectSearchResults(owner, repo, stringToMatch string) map[string]collectedResult {
	collectedResults := make(map[string]collectedResult)
	grepResults := searchCurrentDirectoryUsingGrep(owner, repo, stringToMatch)

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

	return collectedResults
}

func searchCurrentDirectoryUsingGrep(owner, repo, stringToMatch string) []grepResult {
	grepResults := []grepResult{}
	grepOutput, err := exec.Command("grep", "-nri", stringToMatch, ".").Output()
	if err != nil && err.Error() != "exit status 1" {
		logger.Print(owner, repo, "search-grep", "Could not search for", stringToMatch, "using grep:", err.Error())
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
