# go-auctions-client
A Go library and CLI to interact with Filecoin Storage Auctions.

[![Made by Textile](https://img.shields.io/badge/made%20by-Textile-informational.svg)](https://textile.io)
[![Chat on Slack](https://img.shields.io/badge/slack-slack.textile.io-informational.svg)](https://slack.textile.io)
[![standard-readme compliant](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg)](https://github.com/RichardLitt/standard-readme)

Join us on our [public Slack channel](https://slack.textile.io/) for news, discussions, and status updates. [Check out our blog](https://blog.textile.io/) for the latest posts and announcements.

## Table of Contents

- [Installation](#installation)
- [Remote signing CLI](#remote-signing-cli)
- [Remote signing library](#remote-signing-library)
- [Contributing](#contributing)
- [Changelog](#changelog)
- [License](#license)

## Installation

Install Go 1.17, and run:
```bash
make install
```
This will compile the `auc` CLI tool and place it in your `$GOPATH/bin` folder.

Alternatively, you can run:
```bash
make build
```
The `auc` binary will be generated in your current folder.

## Remote signing CLI

The `auc` binary lets you run a remote wallet daemon that your _direct auctions_ API calls 
can target to sign deal proposals resulting from the auctions. Multiple wallet addresses can be 
configured.

The daemon should be publicly reachable, so it's recommended that if you can open firewall ports
and provide open listen addresses, that's highly recommended. As an automatic fallback mechanism,
we make the remote wallet daemon connect with a libp2p relay to have a baked in NAT traversal
solution. The daemon will do its best-effort to keep a healthy connection with the relay automatically.
For best results, please also consider providing direct open ports for your wallet address.

If you want to run a remote wallet that will be used to sign deal proposals:
```bash
$ auc wallet daemon --help
Run a remote wallet signer for auctions

Usage:
  auc wallet daemon [flags]

Flags:
      --auth-token string     Authorization token to validate signing requests
  -h, --help                  help for daemon
      --private-key string    Libp2p private key
      --relay-maddr string    Multiaddress of libp2p relay (default "/ip4/34.105.85.147/tcp/4001/p2p/QmYRDEq8z3Y9hBBAirwMFySuxyCoWwskrD1bxUEYKBiwmU")
      --wallet-keys strings   Wallet address keys; repeatable

Global Flags:
      --log-debug   Enable debug level log (default false)
      --log-json    Enable structured logging
```
A quick explanation of the relevant flags:
- `--auth-token`: Is a string value that will be sent in your _direct auctions_ API calls 
to authenticate with the wallet address. Only requests that provide this auth token will be replied.
- `--relay-maddr`: This an optional flag. By default has value pointing to a libp2p relay we run to help
clients solve NAT problems. If you want to disable this feature, can provide an empty string.
- `--wallet-keys`: Is a comma-separated string value of hex-encoded wallet addresses private keys. (The same format in the output of `lotus wallet export <addr>`).
- `--listen-addresses`: Is a list of multiaddresses to explicitly listen from. Use this flag if you want 
to provide open ports to the wallet address, which will help connectivity.

An example run of this command could be:
```bash
$ auc wallet daemon --debug --auth-token mysecrettk --wallet-keys 7b2254797065223a22626c73222c22507269766174654b6579223a226862702f794666527439514c43716b6d566171415752436f50556777314b776971716e73684e49704e57513d227d
```

If you want to have extra reliability regarding reachability and did port-forwarding configuration in your firewall, here's a possible run of the CLI:
```bash
$ auc wallet daemon --auth-token mysecrettk --debug --wallet-keys 7b2254797065223a22626c73222c22507269766174654b6579223a226862702f794666527439514c43716b6d566171415752436f50556777314b776971716e73684e49704e57513d227d --listen-addresses /ip4/0.0.0.0/tcp/9876
```

The first log lines of the daemon will help understanding which wallet public keys, listen addreses, and relayed addreses are configured:
```bash
2021-09-10T10:07:10.879-0300    INFO    auc     auc/walletCmd.go:48     Loaded wallet: f3rpskqryflc2sqzzzu7j2q6fecrkdkv4p2avpf4kyk5u754he7g6cr2rbpmif7pam5oxbme2oyzot4ry3d74q
2021-09-10T10:07:10.948-0300    INFO    auc     auc/walletCmd.go:92     libp2p peer-id: Qma7rzaZUYNgqSkhgrQ8dmBhPvBhuGk3W7gm1MnoK2Bj9U
2021-09-10T10:07:10.949-0300    INFO    auc     auc/walletCmd.go:94     Listen multiaddr: /ip4/192.168.1.30/tcp/41947
2021-09-10T10:07:10.949-0300    INFO    auc     auc/walletCmd.go:94     Listen multiaddr: /ip4/127.0.0.1/tcp/41947
2021-09-10T10:07:10.949-0300    INFO    auc     auc/walletCmd.go:94     Listen multiaddr: /ip6/::1/tcp/45457
2021-09-10T10:07:10.949-0300    INFO    relaymgr        relaymgr/relaymgr.go:110        connecting with relay...
2021-09-10T10:07:10.955-0300    INFO    relaymgr        relaymgr/relaymgr.go:116        connected with relay
2021-09-10T10:07:10.955-0300    INFO    auc     auc/walletCmd.go:70     Relayed multiaddr: /ip4/140.20.1.1/tcp/9898/p2p/QmfPveoYMS158VbkxNeizZ3ZrDWHb82R28xfkVT9QodcQA/p2p-circuit/Qma7rzaZUYNgqSkhgrQ8dmBhPvBhuGk3W7gm1MnoK2Bj9U
2021-09-10T10:07:30.956-0300    DEBUG   relaymgr        relaymgr/relaymgr.go:104        relay connection is healthy
```
The relay multiaddress circuit is useful to augment your reachable multiaddresses of the remote wallet 


### Remote signing direct-auction API
for the _direct auctions_ API calls.

An example of the body for a direct-auction API call:
```
{
   "payloadCid":"...",
   "pieceCid":"...",
   "pieceSize":...,
   "repFactor":...,
   "deadline":"...",
   "carURL":{...},
   "remoteWallet":{
      # These three fields are mandatory.
      "peerID":"Qma7rzaZUYNgqSkhgrQ8dmBhPvBhuGk3W7gm1MnoK2Bj9U",
      "authToken":"mysecrettk",
      "walletAddr":"f3rpskqryflc2sqzzzu7j2q6fecrkdkv4p2avpf4kyk5u754he7g6cr2rbpmif7pam5oxbme2oyzot4ry3d74q",
      
      # This is an optional but *highly recommended* field.
      # If empty only the relayed address will be used, but for better reliability
      # is encouraged to open ports and provide additional reachable multiaddresses.
      "multiAddrs": ["...", "..."], 
   }
```

## Remote signing library

This repository can also be used as a library, which allows the following use-cases:
- Incorporate a remote wallet in your existing applications.
- Provide your implementation of the [wallet abstraction](https://github.com/textileio/go-auctions-client/blob/main/propsigner/propsigner.go#L36). This can be useful if you want fewer security assumptions, or have the wallet keys in a more constrained environment. The daemon will still be handling the protocol layer of remote signing and deferring signing to your implementation.


## Contributing

Pull requests and bug reports are very welcome ❤️

This repository falls under the Textile [Code of Conduct](./CODE_OF_CONDUCT.md).

Feel free to get in touch by:
-   [Opening an issue](https://github.com/textileio/bidbot/issues/new)
-   Joining the [public Slack channel](https://slack.textile.io/)
-   Sending an email to contact@textile.io

## Changelog

A changelog is published along with each [release](https://github.com/textileio/bidbot/releases).

## License

[MIT](LICENSE)
