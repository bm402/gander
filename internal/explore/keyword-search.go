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
	filename       string
	line           string
	matchedString  string
	exactMatches   int
	similarMatches int
}

func SearchLogsForVariableAssignments(owner, repo, wordlistPath string, threads int) int {
	variableNames := getWordsFromWordlist(wordlistPath)
	logger.Print(owner, repo, "search-variables", "Read", len(variableNames), "variable names from wordlist")

	wg := sync.WaitGroup{}
	variableAssignments := make(chan string, 2*len(variableNames))
	matchesFound := int64(0)

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(owner, repo string, variableAssignments <-chan string) {
			for variableAssignment := range variableAssignments {
				collectedResults := collectSearchResults(owner, repo, variableAssignment)
				if collectedResults.filename != "" {
					atomic.AddInt64(&matchesFound, 1)
					logger.Print(owner, repo, "\033[1;91mmatched-variable\033[0m", "Found", collectedResults.matchedString, "at",
						collectedResults.filename+":"+collectedResults.line+",", "with", collectedResults.exactMatches,
						"exact matches and", collectedResults.similarMatches, "similar matches")
				}
				wg.Done()
			}
		}(owner, repo, variableAssignments)
	}

	// add variable assignments to channel to trigger workers
	for _, variableName := range variableNames {
		wg.Add(1)
		variableAssignments <- variableName + "\\ *[:=]\\ *\\([^\\ &\\\"']\\+\\)"
	}

	// close channel and wait for threads to finish
	close(variableAssignments)
	wg.Wait()

	return int(matchesFound)
}

func SearchLogsForKeywords(owner, repo, wordlistPath string, threads int) int {
	keywords := getWordsFromWordlist(wordlistPath)
	logger.Print(owner, repo, "search-keywords", "Read", len(keywords), "keywords from wordlist")

	wg := sync.WaitGroup{}
	keywordsChan := make(chan string, len(keywords))
	matchesFound := int64(0)

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(owner, repo string, keywordsChan <-chan string) {
			for keyword := range keywordsChan {
				collectedResults := collectSearchResults(owner, repo, keyword)
				if collectedResults.filename != "" {
					atomic.AddInt64(&matchesFound, 1)
					logger.Print(owner, repo, "\033[1;91mmatched-keyword\033[0m", "Found", collectedResults.matchedString, "at",
						collectedResults.filename+":"+collectedResults.line+",", "with", collectedResults.exactMatches,
						"exact matches and", collectedResults.similarMatches, "similar matches")
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

	return int(matchesFound)
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

func collectSearchResults(owner, repo, stringToMatch string) collectedResult {
	grepResults := searchRepoDirectoryUsingGrep(owner, repo, stringToMatch)
	collectedResults := collectedResult{
		exactMatches:   0,
		similarMatches: 0,
	}

	if len(grepResults) > 0 {
		collectedResults.filename = grepResults[0].filename
		collectedResults.line = grepResults[0].line
		collectedResults.matchedString = grepResults[0].matchedString
	} else {
		return collectedResults
	}

	for _, result := range grepResults[1:] {
		if result.matchedString == collectedResults.matchedString {
			collectedResults.exactMatches++
		} else {
			collectedResults.similarMatches++
		}
	}

	return collectedResults
}

func searchRepoDirectoryUsingGrep(owner, repo, stringToMatch string) []grepResult {
	grepResults := []grepResult{}
	grepOutput, err := exec.Command("grep", "-nrio", stringToMatch, owner+"/"+repo).Output()
	if err != nil && err.Error() != "exit status 1" {
		logger.Print(owner, repo, "search-grep", "Could not search for", stringToMatch, "using grep:", err.Error())
	}

	grepOutputLines := strings.Split(string(grepOutput), "\n")
	for _, grepOutputLine := range grepOutputLines {

		// split into filename, line number and matched string (separated by colons but can also contain colons)
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
				partsCount++
			} else {
				line = parts[partsCount]
				partsCount++
				break
			}
		}

		matchedString := strings.TrimSpace(strings.Join(parts[partsCount:], ":"))
		grepResults = append(grepResults, grepResult{
			filename:      filename,
			line:          line,
			matchedString: matchedString,
		})
	}

	return grepResults
}
