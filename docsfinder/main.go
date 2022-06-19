package main

func main() {
	flags := loadFlags()

	downloadGitHubDocs(flags)
}
