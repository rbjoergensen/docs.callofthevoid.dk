package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func downloadGitHubDocs(flags Flags) {
	repositories, err := GetRepositories(flags)
	if err != nil {
		log.Fatal(err)
	}

	// Prepare writer to append to mkdocs config
	f, err := os.OpenFile(flags.MkdocsConfig, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}

	_, err = f.WriteString("\n  - '<b>GitHub</b>':\n")
	if err != nil {
		log.Fatal(err)
	}

	for _, repository := range repositories {
		fmt.Println(repository.FullName)

		rate, err := checkRate(flags)
		for rate.Resources.Search.Remaining <= 1 {
			fmt.Println(fmt.Sprintf("Search rate remaining: %s", strconv.Itoa(rate.Resources.Search.Remaining)))
			time.Sleep(2 * time.Second)
			rate, err = checkRate(flags)
		}

		configSearch, err := CodeSearch(flags, repository, "filename:docs.yml")
		if err != nil {
			log.Fatal(err)
		}

		// GitHub rate limiting throws 403 if there's more than 30 calls per minute
		time.Sleep(2 * time.Second)

		var containsConfig = false
		var docsConfig DocsConfig

		for _, result := range configSearch {
			if result.Path == ".github/docs.yml" {
				content, err := GetContent(flags, result, repository)
				if err != nil {
					log.Fatal(err)
				}

				contentBytes, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					log.Fatal(err)
				}

				err = yaml.Unmarshal(contentBytes, &docsConfig)
				if err != nil {
					log.Fatal(err)
				}

				containsConfig = true
			}
		}

		if containsConfig {
			if docsConfig.NavigatorName == "" {
				log.Fatal(
					fmt.Sprintf(
						"Value of NavigatorName attribute for repository %s is %s",
						repository.Name,
						docsConfig.NavigatorName),
				)
			}

			fmt.Println("  contains-config: true")
			fmt.Println(fmt.Sprintf("  navigator-name:  %s", docsConfig.NavigatorName))
			fmt.Println(fmt.Sprintf("  category:        %s", docsConfig.Category))
		} else if !containsConfig {
			fmt.Println("  contains-config: false")
		}

		mdSearch, err := CodeSearch(flags, repository, "extension:md")
		if err != nil {
			log.Fatal(err)
		}

		// GitHub rate limiting throws 403 if there's more than 30 calls per minute
		time.Sleep(3 * time.Second)

		if len(mdSearch) == 0 {
			continue
		}

		// Value used later to determine if config should be
		// added to the mkdocs.yml for the repository
		var filesAdded int

		// String containing the accumulated configuration
		// that will be added to mkdocs.yml if applicable
		var repositoryNav string

		if containsConfig {
			repositoryNav += fmt.Sprintf(
				"    - '%s':\n",
				docsConfig.NavigatorName)
		}
		if !containsConfig {
			repositoryNav += fmt.Sprintf(
				"    - '%s':\n",
				repository.Name)
		}

		for _, result := range mdSearch {
			if !strings.Contains(result.Path, "/") {
				content, err := GetContent(flags, result, repository)
				if err != nil {
					log.Fatal(err)
				}

				contentBytes, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					log.Fatal(err)
				}

				if len(contentBytes) > flags.MinimumFilesize {
					filesAdded++

					err = os.MkdirAll(fmt.Sprintf("%s/%s", flags.Output, repository.Name), 0755)
					if err != nil {
						log.Fatal(err)
					}

					location := fmt.Sprintf("%s/%s/%s", flags.Output, repository.Name, result.Name)

					err = os.WriteFile(location, contentBytes, 0644)
					if err != nil {
						log.Fatal(err)
					}

					if containsConfig {
						var ruleFound = false
						for _, file := range docsConfig.Files {
							if file.File == result.Path {
								repositoryNav += fmt.Sprintf(
									"      - '%s': 'github/%s/%s'\n",
									file.PrettyName,
									repository.Name,
									result.Name)
							}

							ruleFound = true
						}

						if !ruleFound {
							repositoryNav += fmt.Sprintf(
								"      - '%s': 'github/%s/%s'\n",
								result.Name,
								repository.Name,
								result.Name)
						}
					}

					if !containsConfig {
						repositoryNav += fmt.Sprintf(
							"      - '%s': 'github/%s/%s'\n",
							result.Name,
							repository.Name,
							result.Name)
					}
				}
			}
		}

		repositoryLink := fmt.Sprintf(
			"https://github.com/%s/%s",
			flags.Account,
			repository.Name)

		if containsConfig && docsConfig.IncludeLink {
			repositoryNav += fmt.Sprintf(
				"      - '<span style=\"font-style: italic;\">Link(GitHub)</span>': '%s'\n",
				repositoryLink)
		}

		if !containsConfig {
			repositoryNav += fmt.Sprintf(
				"      - '<span style=\"font-style: italic;\">Link(GitHub)</span>': '%s'\n",
				repositoryLink)
		}

		if filesAdded > 0 {
			_, err = f.WriteString(repositoryNav)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

type DocsConfig struct {
	NavigatorName string     `yaml:"navigatorName"`
	Category      string     `yaml:"category"`
	IncludeLink   bool       `yaml:"includeLink"`
	Files         []DocsFile `yaml:"files"`
}

type DocsFile struct {
	File       string `yaml:"file"`
	PrettyName string `yaml:"prettyName"`
}

func GetContent(flags Flags, file File, repository Repository) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repository.FullName, file.Path)
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("token %s", flags.Token)},
	}

	if flags.Token != "" {
		req.Header = headers
	}

	res, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal(fmt.Sprintf("Statuscode: %s, %s", strconv.Itoa(res.StatusCode), url))
	}

	responseData, err := ioutil.ReadAll(res.Body)

	if err != nil {
		log.Fatal(err)
	}

	var fileContent FileContent

	if err := json.Unmarshal(responseData, &fileContent); err != nil {
		log.Fatal("failed to unmarshal:", err)
	}

	return fileContent.Content, err
}

type FileContent struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

func CodeSearch(flags Flags, repository Repository, query string) ([]File, error) {
	url := fmt.Sprintf("https://api.github.com/search/code?q=repo:%s+%s", repository.FullName, query)
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("token %s", flags.Token)},
	}

	if flags.Token != "" {
		req.Header = headers
	}

	res, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	if res.StatusCode != 200 {
		log.Fatal(fmt.Sprintf("Statuscode: %s, %s", strconv.Itoa(res.StatusCode), url))
	}

	responseData, err := ioutil.ReadAll(res.Body)

	if err != nil {
		log.Fatal(err)
	}

	var files FileRoot

	if err := json.Unmarshal(responseData, &files); err != nil {
		log.Fatal("failed to unmarshal:", err)
	}

	return files.Items, err
}

type FileRoot struct {
	Count int    `json:"total_count"`
	Items []File `json:"items"`
}

type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func GetRepositories(flags Flags) ([]Repository, error) {
	count := 1
	var baseUri = fmt.Sprintf("https://api.github.com/users/%s", flags.Account)
	var repositories []Repository

	if flags.IncludePrivate {
		baseUri = "https://api.github.com/user"
	}

	url := fmt.Sprintf("%s/repos?page=%s&per_page=100", baseUri, strconv.Itoa(count))

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("token %s", flags.Token)},
	}

	if flags.IncludePrivate {
		req.Header = headers
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	responseData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var repositoryList []Repository

	if err := json.Unmarshal(responseData, &repositoryList); err != nil {
		log.Fatal("failed to unmarshal:", err)
	}

	repositories = append(repositories, repositoryList...)

	for len(repositoryList) != 0 {
		count++
		url = fmt.Sprintf("%s/repos?page=%s&per_page=100", baseUri, strconv.Itoa(count))

		client := http.Client{}
		req, err := http.NewRequest("GET", url, nil)

		if flags.IncludePrivate {
			req.Header = headers
		}

		res, err := client.Do(req)
		if err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
		}

		responseData, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		if err := json.Unmarshal(responseData, &repositoryList); err != nil {
			log.Fatal("failed to unmarshal:", err)
		}

		repositories = append(repositories, repositoryList...)
	}

	var filteredRepositories []Repository

	for _, repository := range repositories {
		if flags.SkipArchived && repository.Archived {
			continue
		}

		if !flags.IncludePrivate && repository.Visibility == "private" {
			continue
		}

		if contains(flags.Exclusions, repository.Name) {
			continue
		}

		filteredRepositories = append(filteredRepositories, repository)
	}

	return filteredRepositories, err
}

type Repository struct {
	Name       string `json:"name"`
	FullName   string `json:"full_name"`
	CloneUrl   string `json:"clone_url"`
	Archived   bool   `json:"archived"`
	Visibility string `json:"visibility"`
}
