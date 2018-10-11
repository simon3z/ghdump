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

var GitHubTokenEnvVarName = "GITHUBTOKEN"
var GitHubPasswordEnvVarName = "GITHUBPASSWORD"

var GoogleSheetDateFormat = "01/02/2006 15:04:07"
var GitHubMaxItemsPerPage = 100
var CmdFlagsSinceFormat = "2006-01-02"

var TypePullRequest = "Pull Request"
var TypeIssue = "Issue"

var CmdFlags = struct {
	Username     string
	Password     string
	NoLogin      bool
	TabSeparated bool
	Organization string
	Repository   string
	Since        string
}{}

func init() {
	// By default we retrieve only last month
	since := time.Now().AddDate(0, -1, 0).Format(CmdFlagsSinceFormat)

	flag.StringVar(&CmdFlags.Username, "u", "", "GitHub username")
	flag.BoolVar(&CmdFlags.NoLogin, "n", false, "Do not authenticate (could trigger API rate limits)")
	flag.BoolVar(&CmdFlags.TabSeparated, "t", false, "Use tab-separated output")
	flag.StringVar(&CmdFlags.Organization, "o", "golang", "GitHub owner/organization name")
	flag.StringVar(&CmdFlags.Repository, "r", "go", "GitHub repository name")
	flag.StringVar(&CmdFlags.Since, "s", since, "Retrieve items since specified date")
}

func iterateIssues(client *github.Client, since time.Time, fn func(*github.Issue)) error {
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

func iteratePullRequests(client *github.Client, since time.Time, fn func(*github.PullRequest)) error {
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

func googleSheetHyperlink(value interface{}, link string) string {
	return fmt.Sprintf("=HYPERLINK(\"%s\", \"%v\")", link, value)
}

func gitHubHTTPClient() *http.Client {
	token := os.Getenv(GitHubTokenEnvVarName)

	if len(token) > 0 {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		return oauth2.NewClient(context.Background(), ts)
	}

	if len(CmdFlags.Username) > 0 && len(CmdFlags.Password) > 0 {
		ts := &http.Client{
			Transport: &github.BasicAuthTransport{
				Transport: &http.Transport{},
				Username:  CmdFlags.Username,
				Password:  CmdFlags.Password,
			},
		}
		return ts
	}

	return nil
}

func main() {
	flag.Parse()

	CmdFlags.Password = os.Getenv(GitHubPasswordEnvVarName)

	httpClient := gitHubHTTPClient()
	if httpClient == nil && CmdFlags.NoLogin == false {
		log.Fatal("No authentication could trigger API rate limiting: use authentication or use the flag -n to force.")
	}

	ghClient := github.NewClient(httpClient)

	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	if CmdFlags.TabSeparated {
		w.Comma = '\t'
	}

	sinceDateTime, err := time.Parse(CmdFlagsSinceFormat, CmdFlags.Since)
	if err != nil {
		log.Fatal(err)
	}

	err = iterateIssues(ghClient, sinceDateTime, func(i *github.Issue) {
		w.Write([]string{
			googleSheetHyperlink(*i.User.Login, *i.User.HTMLURL),
			TypeIssue,
			googleSheetHyperlink(*i.Number, *i.HTMLURL),
			*i.Title,
			i.CreatedAt.Format(GoogleSheetDateFormat),
		})
	})

	if err != nil {
		log.Fatal(err)
	}

	err = iteratePullRequests(ghClient, sinceDateTime, func(p *github.PullRequest) {
		w.Write([]string{
			googleSheetHyperlink(*p.User.Login, *p.User.HTMLURL),
			TypePullRequest,
			googleSheetHyperlink(*p.Number, *p.HTMLURL),
			*p.Title,
			p.CreatedAt.Format(GoogleSheetDateFormat),
		})
	})

	if err != nil {
		log.Fatal(err)
	}
}
