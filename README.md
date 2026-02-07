<div align="center">

# 🛠️&nbsp;&nbsp;go-bt

**The go-to Bitcoin Transaction (BT) GoLang library**

<sup>Forked from <a href="https://github.com/libsv/go-bt">libsv/go-bt</a></sup>

<br/>

<a href="https://github.com/bsv-blockchain/go-bt/releases"><img src="https://img.shields.io/github/release-pre/bsv-blockchain/go-bt?include_prereleases&style=flat-square&logo=github&color=black" alt="Release"></a>
<a href="https://golang.org/"><img src="https://img.shields.io/github/go-mod/go-version/bsv-blockchain/go-bt?style=flat-square&logo=go&color=00ADD8" alt="Go Version"></a>
<a href="https://github.com/bsv-blockchain/go-bt/blob/master/LICENSE"><img src="https://img.shields.io/github/license/bsv-blockchain/go-bt?style=flat-square&color=blue" alt="License"></a>

<br/>

<table align="center" border="0">
  <tr>
    <td align="right">
       <code>CI / CD</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/bsv-blockchain/go-bt/actions"><img src="https://img.shields.io/github/actions/workflow/status/bsv-blockchain/go-bt/fortress.yml?branch=master&label=build&logo=github&style=flat-square" alt="Build"></a>
       <a href="https://github.com/bsv-blockchain/go-bt/actions"><img src="https://img.shields.io/github/last-commit/bsv-blockchain/go-bt?style=flat-square&logo=git&logoColor=white&label=last%20update" alt="Last Commit"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Quality</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://goreportcard.com/report/github.com/bsv-blockchain/go-bt/v2"><img src="https://goreportcard.com/badge/github.com/bsv-blockchain/go-bt/v2?style=flat-square" alt="Go Report"></a>
       <a href="https://codecov.io/gh/bsv-blockchain/go-bt"><img src="https://codecov.io/gh/bsv-blockchain/go-bt/branch/master/graph/badge.svg?style=flat-square" alt="Coverage"></a>
    </td>
  </tr>

  <tr>
    <td align="right">
       <code>Security</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://scorecard.dev/viewer/?uri=github.com/bsv-blockchain/go-bt"><img src="https://api.scorecard.dev/projects/github.com/bsv-blockchain/go-bt/badge?style=flat-square" alt="Scorecard"></a>
       <a href=".github/SECURITY.md"><img src="https://img.shields.io/badge/policy-active-success?style=flat-square&logo=security&logoColor=white" alt="Security"></a>
    </td>
    <td align="right">
       &nbsp;&nbsp;&nbsp;&nbsp; <code>Community</code> &nbsp;&nbsp;
    </td>
    <td align="left">
       <a href="https://github.com/bsv-blockchain/go-bt/graphs/contributors"><img src="https://img.shields.io/github/contributors/bsv-blockchain/go-bt?style=flat-square&color=orange" alt="Contributors"></a>
       <a href="https://github.com/sponsors/bsv-blockchain"><img src="https://img.shields.io/badge/sponsor-BSV-181717?style=flat-square&logo=github" alt="Sponsor"></a>
    </td>
  </tr>
</table>

</div>

<br/>
<br/>

<div align="center">

### <code>Project Navigation</code>

</div>

<table align="center">
  <tr>
    <td align="center" width="33%">
       📦&nbsp;<a href="#-installation"><code>Installation</code></a>
    </td>
    <td align="center" width="33%">
       🧪&nbsp;<a href="#-examples--tests"><code>Examples&nbsp;&&nbsp;Tests</code></a>
    </td>
    <td align="center" width="33%">
       📚&nbsp;<a href="#-documentation"><code>Documentation</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
       🤝&nbsp;<a href="#-contributing"><code>Contributing</code></a>
    </td>
    <td align="center">
      🛠️&nbsp;<a href="#-code-standards"><code>Code&nbsp;Standards</code></a>
    </td>
    <td align="center">
      ⚡&nbsp;<a href="#-benchmarks"><code>Benchmarks</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
      🤖&nbsp;<a href="#-ai-usage--assistant-guidelines"><code>AI&nbsp;Usage</code></a>
    </td>
    <td align="center">
       📝&nbsp;<a href="#-license"><code>License</code></a>
    </td>
    <td align="center">
       👥&nbsp;<a href="#-maintainers"><code>Maintainers</code></a>
    </td>
  </tr>
</table>
<br/>

## 🧩 What's Inside
- Full-featured Bitcoin transactions and transaction manipulation/functionality
- Auto-fee calculations for change outputs
- Transaction fee calculation and related checks
- Interfaced signing/unlocking of transaction inputs for easy adaptation/customization and extendability for any use case
- Bitcoin Transaction [Script](bscript) functionality
	- Bitcoin script engine ([interpreter](bscript/interpreter))
	- P2PKH (base58 addresses)
	- Data (OP_RETURN)
	- [BIP276](https://github.com/moneybutton/bips/blob/master/bip-0276.mediawiki)

<br/>

## 📦 Installation

**go-bt** requires a [supported release of Go](https://golang.org/doc/devel/release.html#policy).
```shell script
go get -u github.com/bsv-blockchain/go-bt/v2
```

<br/>

## 📚 Documentation

- **API Reference** – Dive into the godocs at [pkg.go.dev/github.com/bsv-blockchain/go-bt](https://pkg.go.dev/github.com/bsv-blockchain/go-bt)
- **Usage Examples** – Browse practical patterns either the [examples directory](examples)
- **Benchmarks** – Check the latest numbers in the [benchmark results](#benchmark-results)
- **Test Suite** – Review both the [unit tests](tx_test.go) and [fuzz tests](tx_fuzz_test.go) (powered by [`testify`](https://github.com/stretchr/testify))

For more information around the technical aspects of Bitcoin, please see the updated [Bitcoin Wiki](https://wiki.bitcoinsv.io/index.php/Main_Page)

<br/>

<details>
<summary><strong><code>Development Build Commands</code></strong></summary>
<br/>

Get the [MAGE-X](https://github.com/mrz1836/mage-x) build tool for development:
```shell script
go install github.com/mrz1836/mage-x/cmd/magex@latest
```

View all build commands

```bash script
magex help
```

</details>

<details>
<summary><strong>Repository Features</strong></summary>
<br/>

This repository includes 25+ built-in features covering CI/CD, security, code quality, developer experience, and community tooling.

**[View the full Repository Features list →](.github/docs/repository-features.md)**

</details>

<details>
<summary><strong><code>Library Deployment</code></strong></summary>
<br/>

This project uses [goreleaser](https://github.com/goreleaser/goreleaser) for streamlined binary and library deployment to GitHub. To get started, install it via:

```bash
brew install goreleaser
```

The release process is defined in the [.goreleaser.yml](.goreleaser.yml) configuration file.


Then create and push a new Git tag using:

```bash
magex version:bump push=true bump=patch branch=master
```

This process ensures consistent, repeatable releases with properly versioned artifacts and citation metadata.

</details>

<details>
<summary><strong><code>Pre-commit Hooks</code></strong></summary>
<br/>

Set up the Go-Pre-commit System to run the same formatting, linting, and tests defined in [AGENTS.md](.github/AGENTS.md) before every commit:

```bash
go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest
go-pre-commit install
```

The system is configured via modular env files in [`.github/env/`](.github/env/README.md) and provides 17x faster execution than traditional Python-based pre-commit hooks. See the [complete documentation](http://github.com/mrz1836/go-pre-commit) for details.

</details>

<details>
<summary><strong>GitHub Workflows</strong></summary>
<br/>

All workflows are driven by modular configuration in [`.github/env/`](.github/env/README.md) — no YAML editing required.

**[View all workflows and the control center →](.github/docs/workflows.md)**

</details>

<details>
<summary><strong><code>Updating Dependencies</code></strong></summary>
<br/>

To update all dependencies (Go modules, linters, and related tools), run:

```bash
magex deps:update
```

This command ensures all dependencies are brought up to date in a single step, including Go modules and any tools managed by [MAGE-X](https://github.com/mrz1836/mage-x). It is the recommended way to keep your development environment and CI in sync with the latest versions.

</details>

<br/>

## 🧪 Examples & Tests

All unit tests and [examples](examples) run via [GitHub Actions](https://github.com/bsv-blockchain/go-bt/actions) and use [Go version 1.24.x](https://go.dev/doc/go1.24). View the [configuration file](.github/workflows/fortress.yml).

Run all tests (fast):

```bash script
magex test
```

Run all tests with race detector (slower):
```bash script
magex test:race
```

<br/>

## ⚡ Benchmarks

Run the Go [benchmarks](tx_benchmark_test.go):

```bash script
magex bench
```

<br/>

### Benchmark Results

| Benchmark             | Iterations | ns/op | B/op | allocs/op |
|-----------------------|------------|------:|-----:|----------:|
| [Bytes]-toBytesHelper | 4,112,503  | 280.2 | 1024 |         1 |
| [Clone]-clone         | 4,117,903  | 291.4 |  416 |        13 |
| [ShallowClone]-clone  | 5,963,280  | 196.3 |  296 |         8 |

> These benchmarks reflect fast, allocation-free lookups for most retrieval functions, ensuring optimal performance in production environments.
> Performance benchmarks for the core functions in this library, executed on an Apple M1 Max (ARM64).

<br/>

## 🛠️ Code Standards
Read more about this Go project's [code standards](.github/CODE_STANDARDS.md).

<br/>

## 🤖 AI Usage & Assistant Guidelines
Read the [AI Usage & Assistant Guidelines](.github/tech-conventions/ai-compliance.md) for details on how AI is used in this project and how to interact with AI assistants.

<br/>

## 👥 Maintainers
| [<img src="https://github.com/icellan.png" height="50" alt="Siggi" />](https://github.com/icellan) | [<img src="https://github.com/galt-tr.png" height="50" alt="Galt" />](https://github.com/galt-tr) | [<img src="https://github.com/mrz1836.png" height="50" alt="MrZ" />](https://github.com/mrz1836) |
|:--------------------------------------------------------------------------------------------------:|:-------------------------------------------------------------------------------------------------:|:------------------------------------------------------------------------------------------------:|
|                                [Siggi](https://github.com/icellan)                                 |                                [Dylan](https://github.com/galt-tr)                                 |                                [MrZ](https://github.com/mrz1836)                                 |

<br/>

## 🤝 Contributing
View the [contributing guidelines](.github/CONTRIBUTING.md) and please follow the [code of conduct](.github/CODE_OF_CONDUCT.md).

### How can I help?
All kinds of contributions are welcome :raised_hands:!
The most basic way to show your support is to star :star2: the project, or to raise issues :speech_balloon:.

[![Stars](https://img.shields.io/github/stars/bsv-blockchain/go-bt?label=Please%20like%20us&style=social&v=1)](https://github.com/bsv-blockchain/go-bt/stargazers)

<br/>

## 📝 License

[![License](https://img.shields.io/github/license/bsv-blockchain/go-bt?style=flat-square&color=blue)](LICENSE)
