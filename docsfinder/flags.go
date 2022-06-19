package main

import (
	"flag"
	"log"
	"strings"
)

func loadFlags() Flags {
	tokenPtr := flag.String(
		"token",
		"",
		"The token used to access GitHub (*Required)")
	accountPtr := flag.String(
		"account",
		"",
		"The user/organization to target (*Required)")
	skipArchivedPtr := flag.Bool(
		"skip-archived",
		false,
		"Skip archived repositories")
	includePrivatePtr := flag.Bool(
		"include-private",
		false,
		"Include private repositories")
	exclude := flag.String(
		"exclude",
		"",
		"Comma delimited list of repositories to exclude")
	output := flag.String(
		"output",
		"docs",
		"Output directory")
	minimumFilesize := flag.Int(
		"minimum-filesize",
		300,
		"The minimum filesize to download(byte length)")
	mkdocsConfig := flag.String(
		"mkdocs-config",
		"mkdocs.yml",
		"Output directory")

	flag.Parse()

	if *tokenPtr == "" && *includePrivatePtr {
		log.Fatal("--token flag is missing")
	}

	if *accountPtr == "" {
		log.Fatal("--account flag is missing")
	}

	var exclusions []string

	if *exclude != "" {
		exclusions = strings.Split(*exclude, ",")
	}

	return Flags{
		Token:           *tokenPtr,
		Account:         *accountPtr,
		SkipArchived:    *skipArchivedPtr,
		IncludePrivate:  *includePrivatePtr,
		Exclusions:      exclusions,
		Output:          *output,
		MinimumFilesize: *minimumFilesize,
		MkdocsConfig:    *mkdocsConfig,
	}
}

type Flags struct {
	Token           string
	Account         string
	SkipArchived    bool
	IncludePrivate  bool
	Exclusions      []string
	Output          string
	MinimumFilesize int
	MkdocsConfig    string
}
