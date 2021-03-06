package explore

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/bm402/gander/internal/logger"
)

type grepResult struct {
	filename      string
	line          string
	matchedString string
}

type condensedResultByFileKey struct {
	filename      string
	matchedString string
}

type condensedResultByFileValue struct {
	line        string
	occurrences int
}

type CollectedResult struct {
	Filename    string
	Line        string
	Files       int
	Occurrences int
	IsCondensed bool
}

func SearchLogsForVariableAssignments(owner, repo, wordlistPath string, threads int) map[string]CollectedResult {
	variableNames := getWordsFromWordlist(wordlistPath)
	logger.Print(owner, repo, "search-variables", "Read", len(variableNames), "variable names from wordlist")

	wg := sync.WaitGroup{}
	variableAssignments := make(chan string, 2*len(variableNames))
	globalCollectedResults := make(map[string]CollectedResult)
	mutex := &sync.Mutex{}

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(owner, repo string, variableAssignments <-chan string) {
			for variableAssignment := range variableAssignments {
				collectedResults := collectSearchResults(owner, repo, variableAssignment)
				for matchedString, collectedResult := range collectedResults {
					mutex.Lock()
					globalCollectedResults[matchedString] = collectedResult
					mutex.Unlock()
					if collectedResult.IsCondensed {
						logger.Print(owner, repo, "\033[1;91mmatched-variable\033[0m", "Found", matchedString, "at",
							collectedResult.Filename+":"+collectedResult.Line+", with", collectedResult.Occurrences,
							"similar occurrences (probably randomly generated)")
					} else {
						logger.Print(owner, repo, "\033[1;91mmatched-variable\033[0m", "Found", matchedString, "at",
							collectedResult.Filename+":"+collectedResult.Line+", with", collectedResult.Occurrences,
							"occurrences in", collectedResult.Files, "files")
					}
				}
				wg.Done()
			}
		}(owner, repo, variableAssignments)
	}

	// add variable assignments to channel to trigger workers
	for _, variableName := range variableNames {
		wg.Add(1)
		variableAssignments <- "[^\\ ?&]*" + variableName + "\\ *[:=]\\ *\\([^\\ &]\\+\\)"
	}

	// close channel and wait for threads to finish
	close(variableAssignments)
	wg.Wait()

	return globalCollectedResults
}

func SearchLogsForKeywords(owner, repo, wordlistPath string, threads int) map[string]CollectedResult {
	keywords := getWordsFromWordlist(wordlistPath)
	logger.Print(owner, repo, "search-keywords", "Read", len(keywords), "keywords from wordlist")

	wg := sync.WaitGroup{}
	keywordsChan := make(chan string, len(keywords))
	globalCollectedResults := make(map[string]CollectedResult)
	mutex := &sync.Mutex{}

	// create worker threads
	for i := 0; i < threads; i++ {
		go func(owner, repo string, keywordsChan <-chan string) {
			for keyword := range keywordsChan {
				collectedResults := collectSearchResults(owner, repo, keyword)
				for matchedString, collectedResult := range collectedResults {
					mutex.Lock()
					globalCollectedResults[matchedString] = collectedResult
					mutex.Unlock()
					if collectedResult.IsCondensed {
						logger.Print(owner, repo, "\033[1;91mmatched-keyword\033[0m", "Found", matchedString, "at",
							collectedResult.Filename+":"+collectedResult.Line+", with", collectedResult.Occurrences,
							"similar occurrences (probably randomly generated)")
					} else {
						logger.Print(owner, repo, "\033[1;91mmatched-keyword\033[0m", "Found", matchedString, "at",
							collectedResult.Filename+":"+collectedResult.Line+", with", collectedResult.Occurrences,
							"occurrences in", collectedResult.Files, "files")
					}
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

	return globalCollectedResults
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

func collectSearchResults(owner, repo, stringToMatch string) map[string]CollectedResult {
	grepResults := searchRepoDirectoryUsingGrep(owner, repo, stringToMatch)

	// condense grep results into files: matchedString, filename => line (first occurrence), occurrences
	condensedResultsByFile := make(map[condensedResultByFileKey]condensedResultByFileValue)
	for _, grepResult := range grepResults {

		// skip censored and blank variable assignments
		if strings.Contains(stringToMatch, "[:=]") && isVariableAssignmentBlankOrCensored(grepResult.matchedString) {
			continue
		}

		fileMatchKey := condensedResultByFileKey{
			filename:      grepResult.filename,
			matchedString: grepResult.matchedString,
		}
		if existingFileMatchOccurrences, exists := condensedResultsByFile[fileMatchKey]; exists {
			updatedFileMatchOccurrences := existingFileMatchOccurrences
			updatedFileMatchOccurrences.occurrences++
			condensedResultsByFile[fileMatchKey] = updatedFileMatchOccurrences
		} else {
			condensedResultsByFile[fileMatchKey] = condensedResultByFileValue{
				line:        grepResult.line,
				occurrences: 1,
			}
		}
	}

	// condense file occurrences into global occurrences: matchedString => filename (first occurrence), line (first occurrence), files, occurrences
	collectedResults := make(map[string]CollectedResult)
	for fileMatch, occurrences := range condensedResultsByFile {
		if existingCollectedResult, exists := collectedResults[fileMatch.matchedString]; exists {
			updatedCollectedResult := existingCollectedResult
			updatedCollectedResult.Files++
			updatedCollectedResult.Occurrences += occurrences.occurrences
			collectedResults[fileMatch.matchedString] = updatedCollectedResult
		} else {
			collectedResults[fileMatch.matchedString] = CollectedResult{
				Filename:    fileMatch.filename,
				Line:        occurrences.line,
				Files:       1,
				Occurrences: occurrences.occurrences,
				IsCondensed: false,
			}
		}
	}

	// if more than n matchedStrings, combine occurrences in only a single file to one log entry as they are probably randomly generated
	if len(collectedResults) > DUPLICATE_RESULTS_THRESHOLD {
		matchedStringsToDelete := []string{}
		firstMatchedString := ""
		condensedResult := CollectedResult{
			IsCondensed: true,
		}

		for matchedString, result := range collectedResults {
			if result.Files < 2 {
				matchedStringsToDelete = append(matchedStringsToDelete, matchedString)
				if firstMatchedString != "" {
					condensedResult.Occurrences += result.Occurrences
				} else {
					firstMatchedString = matchedString
					condensedResult.Filename = result.Filename
					condensedResult.Line = result.Line
					condensedResult.Occurrences = result.Occurrences
				}
			}
		}

		for _, matchedStringToDelete := range matchedStringsToDelete {
			delete(collectedResults, matchedStringToDelete)
		}

		collectedResults[firstMatchedString] = condensedResult
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

func isVariableAssignmentBlankOrCensored(matchedString string) bool {
	if strings.Index(matchedString, ":") == len(matchedString)-1 && !strings.Contains(matchedString, "=") {
		return true
	}
	if strings.Index(matchedString, "=") == len(matchedString)-1 && !strings.Contains(matchedString, ":") {
		return true
	}
	if len(matchedString) >= 3 && matchedString[len(matchedString)-3:] == "***" {
		return true
	}
	return false
}
