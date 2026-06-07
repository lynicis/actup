<!-- Improved compatibility of back to top link: See: https://github.com/othneildrew/Best-README-Template/pull/73 -->
<a id="readme-top"></a>

<!-- PROJECT SHIELDS -->
<!--
*** I'm using markdown "reference style" links for readability.
*** Reference links are enclosed in brackets [ ] instead of parentheses ( ).
*** See the bottom of this document for the declaration of the reference variables
*** for contributors-url, forks-url, etc. This is an optional, concise syntax you may use.
*** https://www.markdownguide.org/basic-syntax/#reference-style-links
-->
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![Go Version][go-shield]][go-url]
[![License][license-shield]][license-url]



<h1 align="center">actup</h1>

<p align="center">
  Upgrade GitHub Actions versions interactively from your terminal
  <br />
  <a href="https://github.com/lynicis/actup"><strong>Explore the docs »</strong></a>
  <br />
  <br />
  <a href="https://github.com/lynicis/actup/issues/new?labels=bug&template=bug-report---.md">Report Bug</a>
  ·
  <a href="https://github.com/lynicis/actup/issues/new?labels=enhancement&template=feature-request---.md">Request Feature</a>
</p>

<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
    <li><a href="#acknowledgments">Acknowledgments</a></li>
  </ol>
</details>



<!-- ABOUT THE PROJECT -->
## About The Project

`actup` is a CLI tool that scans your GitHub Actions workflow files (`.github/workflows/*.yml`) and upgrades action versions to their latest semver tags. It provides both an interactive terminal UI (TUI) powered by Bubble Tea for cherry-picking upgrades, and a non-interactive mode for CI/automation.

Key features:
- 🔍 **Automatic discovery** of workflow files in `.github/workflows/`
- 📊 **Interactive TUI** with a checkbox list to select which actions to upgrade
- 🚀 **Non-interactive mode** (`--no-tui`) for automated upgrades
- 👀 **Dry-run support** (`--dry-run`) to preview changes without writing files
- 🔒 **Atomic file edits** via temp-file + rename to prevent corruption
- 🏷️ **Semver-aware** — fetches and sorts tags by semantic versioning
- ⚡ **Concurrent API requests** with built-in rate-limiting and caching

<p align="right">(<a href="#readme-top">back to top</a>)</p>



### Built With

* [![Go][Go-shield]][Go-url]
* [![Cobra][Cobra-shield]][Cobra-url]
* [![Bubble Tea][BubbleTea-shield]][BubbleTea-url]
* [![go-github][go-github-shield]][go-github-url]

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- GETTING STARTED -->
## Getting Started

To get a local copy up and running follow these simple steps.

### Prerequisites

You need Go installed on your machine. The project builds with Go 1.22 or later.

* Go
  ```sh
  # macOS
  brew install go

  # Ubuntu/Debian
  sudo apt install golang-go

  # Or download from https://go.dev/dl/
  ```

### Installation

#### Option 1: Install via `go install`

```sh
go install github.com/lynicis/actup@latest
```

#### Option 2: Build from source

1. Clone the repo
   ```sh
   git clone https://github.com/lynicis/actup.git
   cd actup
   ```
2. Build the binary
   ```sh
   make build
   ```
3. (Optional) Install to your `$GOPATH/bin`
   ```sh
   make install
   ```

#### Option 3: Download a release

Grab a pre-built binary for Linux, macOS, or Windows from the [Releases](https://github.com/lynicis/actup/releases) page.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- USAGE EXAMPLES -->
## Usage

Run `actup` from the root of any repository containing GitHub Actions workflows:

```sh
# Interactive mode (default) — opens a TUI to select upgrades
actup

# Non-interactive mode — upgrades everything automatically
actup --no-tui

# Preview changes without writing files
actup --dry-run

# Scan custom paths
actup -p ./my-workflows -p ./another-workflows

# Provide a GitHub token for higher rate limits
actup -t $GITHUB_TOKEN
# or
export GITHUB_TOKEN=ghp_xxx
actup
```

### Interactive TUI Controls

| Key | Action |
|-----|--------|
| `Space` | Toggle selection of an action |
| `a` | Select all upgradable actions |
| `n` | Deselect all |
| `Enter` | Apply selected upgrades |
| `q` / `Ctrl+C` | Quit |

### Example Output

```
  actup — 5 actions found across 3 files

  [✓] actions/checkout@v3       → v4 (3 files)
  [✓] actions/setup-go@v4       → v5 (2 files)
  [⏭] actions/cache@v4           (up to date)
  [✓] golangci/golangci-lint-action@v3 → v6 (1 file)
  [⚠] some-org/some-action@v1    (API error)

  [space] toggle  [a] all  [n] none  [enter] apply  [q] quit
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- ROADMAP -->
## Roadmap

- [x] Interactive TUI with checklist selection
- [x] Non-interactive (`--no-tui`) mode
- [x] Dry-run support
- [x] Concurrent GitHub API calls with rate-limit awareness
- [x] Cross-platform builds (Linux, macOS, Windows)
- [ ] Add support for pinning to specific major versions (`--major`)
- [ ] Config file support (`.actup.yaml`)
- [ ] Integration with `dependabot`-style grouped updates
- [ ] Pre-upgrade hooks / custom validation

See the [open issues](https://github.com/lynicis/actup/issues) for a full list of proposed features (and known issues).

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- CONTRIBUTING -->
## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".
Don't forget to give the project a star! Thanks again!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'feat: add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

Please make sure your code passes the existing tests and lint checks:

```sh
make test
make lint
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- LICENSE -->
## License

Distributed under the MIT License. See `LICENSE` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- CONTACT -->
## Contact

Project Link: [https://github.com/lynicis/actup](https://github.com/lynicis/actup)

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- ACKNOWLEDGMENTS -->
## Acknowledgments

* [Bubble Tea](https://github.com/charmbracelet/bubbletea) — the TUI framework that powers the interactive interface
* [Cobra](https://github.com/spf13/cobra) — CLI framework for Go
* [go-github](https://github.com/google/go-github) — GitHub API client library
* [Best-README-Template](https://github.com/othneildrew/Best-README-Template) — README template inspiration

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[contributors-shield]: https://img.shields.io/github/contributors/lynicis/actup.svg?style=for-the-badge
[contributors-url]: https://github.com/lynicis/actup/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/lynicis/actup.svg?style=for-the-badge
[forks-url]: https://github.com/lynicis/actup/network/members
[stars-shield]: https://img.shields.io/github/stars/lynicis/actup.svg?style=for-the-badge
[stars-url]: https://github.com/lynicis/actup/stargazers
[issues-shield]: https://img.shields.io/github/issues/lynicis/actup.svg?style=for-the-badge
[issues-url]: https://github.com/lynicis/actup/issues
[go-shield]: https://img.shields.io/github/go-mod/go-version/lynicis/actup?style=for-the-badge
[go-url]: https://go.dev/
[license-shield]: https://img.shields.io/github/license/lynicis/actup.svg?style=for-the-badge
[license-url]: https://github.com/lynicis/actup/blob/main/LICENSE
[Go-shield]: https://img.shields.io/badge/go-00ADD8?style=for-the-badge&logo=go&logoColor=white
[Go-url]: https://go.dev/
[Cobra-shield]: https://img.shields.io/badge/Cobra-3C3C3C?style=for-the-badge
[Cobra-url]: https://github.com/spf13/cobra
[BubbleTea-shield]: https://img.shields.io/badge/Bubble%20Tea-ff75cd?style=for-the-badge
[BubbleTea-url]: https://github.com/charmbracelet/bubbletea
[go-github-shield]: https://img.shields.io/badge/go--github-181717?style=for-the-badge&logo=github&logoColor=white
[go-github-url]: https://github.com/google/go-github
