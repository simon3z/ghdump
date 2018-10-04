package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var GitHubEnvironmentVarName = "GITHUBTOKEN"
var GoogleSheetDateFormat = "01/02/2006 15:04:07"
var GitHubMaxItemsPerPage = 100
var CmdFlagsSinceFormat = "2006-01-02"

var TypePullRequest = "Pull Request"
var TypeIssue = "Issue"

var CmdFlags = struct {
	TabSeparated bool
	Organization string
	Repository   string
	Since        string
}{}

func init() {
	// By default we retrieve only last month
	since := time.Now().AddDate(0, -1, 0).Format(CmdFlagsSinceFormat)

	flag.BoolVar(&CmdFlags.TabSeparated, "t", false, "Use tab-separated output")
	flag.StringVar(&CmdFlags.Organization, "o", "golang", "GitHub Owner Name")
	flag.StringVar(&CmdFlags.Repository, "r", "go", "GitHub Repository Name")
	flag.StringVar(&CmdFlags.Since, "s", since, "Retrieve items since specified date")
}

func IterateIssues(client *github.Client, since time.Time, fn func(*github.Issue)) error {
	options := github.IssueListByRepoOptions{
		Direction:   "desc",
		Sort:        "created",
		State:       "all",
		ListOptions: github.ListOptions{PerPage: GitHubMaxItemsPerPage},
	}

	for {
		issues, response, err := client.Issues.ListByRepo(context.Background(), CmdFlags.Organization, CmdFlags.Repository, &options)
		if err != nil {
			return err
		}

		for _, i := range issues {
			if i.CreatedAt.Before(since) {
				return nil
			}

			fn(i)
		}

		if response.NextPage == 0 {
			break
		}

		options.Page = response.NextPage
	}

	return nil
}

func IteratePullRequests(client *github.Client, since time.Time, fn func(*github.PullRequest)) error {
	options := github.PullRequestListOptions{
		Direction:   "desc",
		Sort:        "created",
		State:       "all",
		ListOptions: github.ListOptions{PerPage: GitHubMaxItemsPerPage},
	}

	for {
		pullrequests, response, err := client.PullRequests.List(context.Background(), CmdFlags.Organization, CmdFlags.Repository, &options)
		if err != nil {
			return err
		}

		for _, i := range pullrequests {
			if i.CreatedAt.Before(since) {
				return nil
			}

			fn(i)
		}

		if response.NextPage == 0 {
			break
		}

		options.Page = response.NextPage
	}

	return nil
}

func GoogleSheetHyperlink(value interface{}, link string) string {
	return fmt.Sprintf("=HYPERLINK(\"%s\", \"%v\")", link, value)
}

func GitHubHTTPClient() *http.Client {
	token := os.Getenv(GitHubEnvironmentVarName)

	if len(token) == 0 {
		return nil
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

	return oauth2.NewClient(context.Background(), ts)
}

func main() {
	flag.Parse()

	sinceDateTime, err := time.Parse(CmdFlagsSinceFormat, CmdFlags.Since)
	if err != nil {
		log.Fatal(err)
	}

	client := github.NewClient(GitHubHTTPClient())

	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	if CmdFlags.TabSeparated {
		w.Comma = '\t'
	}

	err = IterateIssues(client, sinceDateTime, func(i *github.Issue) {
		w.Write([]string{
			GoogleSheetHyperlink(*i.User.Login, *i.User.HTMLURL),
			TypeIssue,
			GoogleSheetHyperlink(*i.Number, *i.HTMLURL),
			*i.Title,
			i.CreatedAt.Format(GoogleSheetDateFormat),
		})
	})

	if err != nil {
		log.Fatal(err)
	}

	err = IteratePullRequests(client, sinceDateTime, func(p *github.PullRequest) {
		w.Write([]string{
			GoogleSheetHyperlink(*p.User.Login, *p.User.HTMLURL),
			TypePullRequest,
			GoogleSheetHyperlink(*p.Number, *p.HTMLURL),
			*p.Title,
			p.CreatedAt.Format(GoogleSheetDateFormat),
		})
	})

	if err != nil {
		log.Fatal(err)
	}
}
