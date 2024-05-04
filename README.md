# gh-flox

Query github for flox related items.

# Usage

gh-flox stars - Number of :star: found on flox/flox on GitHub
gh-flox repos -v - List the repositories containing a `.flox` directory.
gh-flox repos -f - Count number of repos with a `.flox` including those owned by flox and employees
gh-flox repos -f -v - List the repos with a `.flox` including those owned by flox and employees

# Configuration

To run with slack formatting, set `SLACK_MODE=1`. Otherwise, plain text is assumed.

# Development

`flox activate`
`make`

# Deployment

`make ready` ships to the hubot server if you're all set up. 

# License
MIT
