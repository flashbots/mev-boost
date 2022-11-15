# Security Policy

We appreciate any contributions, responsible disclosures and will make every
effort to acknowledge your contributions.

## Supported Versions

Please see [Releases](https://github.com/flashbots/mev-boost/releases).
Generally it is recommended to use
the [latest release](https://github.com/flashbots/mev-boost/releases/latest).

## Reporting a Vulnerability

To report a vulnerability, please email security@flashbots.net and provide all
the necessary details to reproduce it, such as:

- Release version
- Operating System
- Consensus / Execution client combination and version
- Network (Mainnet or other testnet)

Please include the steps to reproduce it using as much detail as possible with
the corresponding logs from `mev-boost` and/or logs from the consensus/execution
client.

Once we have received your bug report, we will try to reproduce it and provide a
more detailed response. When the reported bug has been successfully reproduced,
the team will work on a fix.

## Bug Bounty Program

To incentive bug reports, there is a bug bounty program. You can receive a
bounty (up to $25k USD) depending on the bug's severity. Severity is based on
impact and likelihood. If a bug is high impact but low likelihood, it will have
a lower severity than a bug with a high impact and high likelihood.

| Severity |     Maximum | Example                                                                |
|----------|------------:|------------------------------------------------------------------------|
| Low      |  $1,000 USD | A bug that causes mev-boost to skip a bid.                             |
| Medium   |  $5,000 USD | From a builder message, can cause mev-boost to go offline.             |
| High     | $12,500 USD | From a builder message, can cause a connected validator to go offline. |
| Critical | $25,000 USD | From a builder message, can remotely access arbitrary files on host.   |

### Scope

Bugs that affect the security of the Ethereum protocol in the `mev-boost`
and `mev-boost-relay` repositories are in scope. Bugs in third-party
dependencies are not in scope unless they result in a bug in `mev-boost` with
demonstrable security impact.
