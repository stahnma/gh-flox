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

## Hand edits

Sometimes, a repository has installations instruction for flox, but not in the
primary README. In this case, you can add the slug `owner/repo` to the
`additional_repos.json` file and this will be picked up by the `readmes` sub
command. This is only read at __build time__, so you need to rebuild the
project if that file changes. This is to assist in running it as a lambda
function.

# Development

`flox activate`

`make`

# Deployment

`make ready` ships to the hubot server if you're all set up.

# Code Flow

## High-Level Architecture

```mermaid
graph TD
    A[main.go: main] -->|LAMBDA_TASK_ROOT set| B[Lambda Mode]
    A -->|LAMBDA_TASK_ROOT unset| C[CLI Mode]

    B --> B1[lambdaHandler]
    B1 --> B2[Force 'export' command]
    B2 --> B3[Capture stdout]
    B3 --> B4[Upload JSON to S3]

    C --> D[cobra rootCmd.Execute]
    D --> E1[repos]
    D --> E2[stars]
    D --> E3[readmes]
    D --> E4[floxindex]
    D --> E5[export]
    D --> E6[version]
    D --> E7[clearcache]
    D --> E8[download-manifests]

    C -->|after execution| F[Save cache to /tmp/cache.gob]
```

## Command Execution Flow

```mermaid
flowchart TD
    subgraph init["init()"]
        I1[Register all subcommands with cobra] --> I2[Load cache from /tmp/cache.gob]
        I2 --> I3[Configure Viper: SLACK_MODE]
        I3 --> I4["Parse additional_repos.json (go:embed)"]
    end

    subgraph main["main()"]
        M1{LAMBDA_TASK_ROOT?} -->|yes| LAMBDA
        M1 -->|no| CLI
    end

    init --> main

    subgraph LAMBDA["Lambda Path"]
        L1[Set os.Args to 'export'] --> L2[Redirect stdout to pipe]
        L2 --> L3[Execute rootCmd]
        L3 --> L4[Read captured output]
        L4 --> L5[Connect to S3]
        L5 --> L6["Upload as {date}.json"]
    end

    subgraph CLI["CLI Path"]
        C1[rootCmd.Execute] --> C2{Which subcommand?}
        C2 --> CMD
    end

    subgraph CMD["Command Handlers"]
        R1[runReposCommand]
        R2[runStarsCommand]
        R3[runReadmesCommand]
        R4[runFloxIndexCommand]
        R5[runExportJSONCommand]
        R6[runVersionCommand]
        R7[runClearCacheCommand]
        R8[runDownloadManifestsCommand]
    end

    CLI --> SAVE[saveCacheToFile]
```

## Core Search Functions

```mermaid
flowchart TD
    subgraph findManifest["findAllFloxManifestRepos(showFull, verbose)"]
        FM1{Cache hit?} -->|yes| FM_RET[Return cached results]
        FM1 -->|no| FM2["GitHub Search:\n'.flox/env/manifest.toml in:path'"]
        FM2 --> FM3[Paginate results]
        FM3 --> FM4[Deduplicate repos via map]
        FM4 --> FM5{showFull?}
        FM5 -->|no| FM6["Filter: isOrgMember(owner, 'flox')\nExclude 'flox' & 'flox-examples' orgs"]
        FM5 -->|yes| FM7[Keep all repos]
        FM6 --> FM8{verbose?}
        FM7 --> FM8
        FM8 -->|yes| FM9[Fetch star counts per repo]
        FM8 -->|no| FM10[Skip star counts]
        FM9 --> FM11[Sort alphabetically]
        FM10 --> FM11
        FM11 --> FM12[Cache results]
        FM12 --> FM_RET
    end

    subgraph findReadme["findAllFloxReadmeRepos(showFull, verbose)"]
        FR1{Cache hit?} -->|yes| FR_RET[Return cached results]
        FR1 -->|no| FR2["GitHub Search:\n'\"flox install\" in:file filename:README'"]
        FR2 --> FR3[Paginate results]
        FR3 --> FR4[Deduplicate repos via map]
        FR4 --> FR5{showFull?}
        FR5 -->|no| FR6["Filter: isOrgMember(owner, 'flox')\nExclude 'flox' & 'flox-examples' orgs"]
        FR5 -->|yes| FR7[Keep all repos]
        FR6 --> FR8[Merge additional_repos.json entries]
        FR7 --> FR8
        FR8 --> FR9{verbose?}
        FR9 -->|yes| FR10[Fetch star counts per repo]
        FR9 -->|no| FR11[Skip star counts]
        FR10 --> FR12[Sort alphabetically]
        FR11 --> FR12
        FR12 --> FR13[Cache results]
        FR13 --> FR_RET
    end
```

## Command Detail: repos

```mermaid
flowchart LR
    A[runReposCommand] --> B["Parse flags:\n-v/--verbose\n-f/--full"]
    B --> C[findAllFloxManifestRepos]
    C --> D{SLACK_MODE?}
    D -->|yes| E["Bold count + code block list"]
    D -->|no| F["Plain text count + list"]
```

## Command Detail: readmes

```mermaid
flowchart LR
    A[runReadmesCommand] --> B["Parse flags:\n-v/--verbose\n-f/--full"]
    B --> C[findAllFloxReadmeRepos]
    C --> D{SLACK_MODE?}
    D -->|yes| E["Bold count + star2 emoji\n+ code block list"]
    D -->|no| F["Plain text count + list"]
```

## Command Detail: export (JSON)

```mermaid
flowchart TD
    A[runExportJSONCommand] --> B[Get current date]
    B --> C["findAllFloxManifestRepos(verbose=true)"]
    C --> D["Build RepoInfo[]\ntype='dotflox'"]
    D --> E["findAllFloxReadmeRepos(verbose=true)"]
    E --> F["Build RepoInfo[]\ntype='readme'"]
    F --> G[Merge both arrays]
    G --> H[Marshal to JSON]
    H --> I[Print to stdout]
```

## Command Detail: floxindex

```mermaid
flowchart TD
    A[runFloxIndexCommand] --> B[calculateFloxIndex]
    B --> C["findAllFloxManifestRepos(verbose=true)"]
    C --> D[Sum manifest repo stars]
    D --> E["findAllFloxReadmeRepos(verbose=true)"]
    E --> F[Sum readme repo stars]
    F --> G[Sum additional repo stars]
    G --> H["Print total 'Flox Index'"]
```

## Command Detail: download-manifests

```mermaid
flowchart TD
    A[runDownloadManifestsCommand] --> B["findAllFloxManifestRepos(full=false)"]
    B --> C["Create 'manifests/' directory"]
    C --> D[For each repo]
    D --> E["fetchManifestFile(owner, repo)"]
    E --> F["GitHub Search:\n'manifest.toml repo:owner/repo path:.flox/env'"]
    F --> G[Get file path from result]
    G --> H["HTTP GET raw.githubusercontent.com\n/{owner}/{repo}/HEAD/{path}"]
    H --> I["Save as manifests/{owner}_{repo}_manifest.toml"]
```

## Cache System

```mermaid
flowchart TD
    subgraph lifecycle["Cache Lifecycle"]
        CL1["init(): loadCacheFromFile()"] --> CL2["In-memory go-cache\n4hr expiry, 6hr cleanup"]
        CL2 --> CL3["main() exit: saveCacheToFile()"]
        CL3 --> CL4["/tmp/cache.gob (GOB encoding)"]
        CL4 -.->|next run| CL1
    end

    subgraph keys["Cache Keys"]
        K1["floxManifestRepos:{showFull}:{verbose}"]
        K2["floxReadmeRepos:{showFull}:{verbose}"]
        K3["starCount:{owner}/{repo}"]
    end

    subgraph flags["Cache Control"]
        F1["--no-cache flag"] --> F2[Skip cache reads]
        F1 --> F3[Skip cache writes]
        F1 --> F4[Skip save on exit]
        F5[clearcache command] --> F6[Flush all + save empty]
    end
```

## GitHub API Interactions

```mermaid
flowchart LR
    subgraph auth["Authentication"]
        A1[GITHUB_TOKEN env var] --> A2[OAuth2 token source]
        A2 --> A3[go-github client]
    end

    subgraph api["API Calls"]
        A3 --> B1["Search.Code()\n- manifest search\n- readme search\n- manifest file search"]
        A3 --> B2["Organizations.IsMember()\n- org filtering"]
        A3 --> B3["Repositories.Get()\n- star counts"]
    end

    subgraph external["External HTTP"]
        E1["raw.githubusercontent.com\n- manifest downloads"]
    end
```

## Data Structures

```mermaid
classDiagram
    class RepoInfo {
        +string Date
        +string Repository
        +string Type
        +int StarCount
    }

    class CacheSystem {
        +cache.Cache resultCache
        +loadCacheFromFile() error
        +saveCacheToFile() error
    }

    class GlobalState {
        +string GitSHA
        +string GitDirty
        +bool slackMode
        +bool noCache
        +[]string additionalRepos
        +map orgMemberCache
    }

    RepoInfo --> GlobalState : created by export cmd
    CacheSystem --> GlobalState : managed via flags
```

## Deployment Modes

```mermaid
graph LR
    subgraph build["Build Targets (Makefile)"]
        B1["make local\n(current OS)"]
        B2["make ready\n(Linux → bot server)"]
        B3["make lambda\n(Linux ARM64 → AWS)"]
    end

    B1 --> D1["gh-flox binary\n(gh CLI extension)"]
    B2 --> D2["/var/lib/goldiflox/shell/\n(production server)"]
    B3 --> D3["AWS Lambda\n(scheduled export → S3)"]
```

# License
MIT
