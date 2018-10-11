# GitHub Dump Tool

Tool for CSV/TSV dumping GitHub Issues and Pull Requests.

# Installation

    $ go get github.com/simon3z/ghdump

# Running the Tool

    $ ghdump -h
      -o string
            GitHub Owner Name (default "golang")
      -r string
            GitHub Repository Name (default "go")
      -s string
            Retrieve items since specified date (default "2018-09-04")
      -t    Use tab-separated output

    $ GITHUBTOKEN=<token> ghdump -t -o golang -r go
    ...

    $ GITHUBPASSWORD=<password> ghdump -t -u <username> -o golang -r go
    ...
