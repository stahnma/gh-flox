# gh-flox

Query github for flox related items.

# Usage

`gh-flox stars` - Number of :star: found on flox/flox on GitHub

`gh-flox repos -v` - List the repositories containing a `.flox` directory.

`gh-flox repos -f` - Count number of repos with a `.flox` including those owned by flox and employees

`gh-flox repos -f -v` - List the repos with a `.flox` including those owned by flox and employees

`gh-flox readmes` - Count number of repos with `flox install` string in their README

`gh-flox readmes -v` - List repos that have a README with `flox install` string in them

`gh-flox readmes -f` - Count repos with README with `flox install` including those owned by flox and employees

`gh-flox readmes -f -v` - List repos that have a README with `flox install` string in them including those owned by flox and employees

`gh-flox clearcache` - clear out the local cache

`gh-flox version` - get version of `gh-flox`

`gh-flox floxindex` - get the sum of all stars for repos scoped with `readmes` and `repos` subcommands.

`gh-flox export` - Export to JSON

# Configuration

To run with slack formatting, set `SLACK_MODE=1`. Otherwise, plain text is assumed.


  * `GITHUB_TOKEN` - required to query GitHub API
  * `S3_BUCKET_NAME` - optional, only needed when running as a lambda
  * `S3_OBJECT_KEY` - optional, only needed when running as a lambda 
  * `AWS_REGION` - optional, only needed when running as a lambda

# Development

`flox activate`

`make`

# Deployment

`make ready` ships to the hubot server if you're all set up. 

# License
MIT
