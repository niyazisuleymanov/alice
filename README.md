# alice

alice is a BitTorrent client written from scratch in Go.

# Features

- [The BitTorrent Protocol Specification](https://www.bittorrent.org/beps/bep_0003.html)
- [UDP Tracker Protocol for BitTorrent](https://www.bittorrent.org/beps/bep_0015.html)
- [DHT Protocol](https://www.bittorrent.org/beps/bep_0005.html)
- [Multitracker Metadata Extension](https://www.bittorrent.org/beps/bep_0012.html)

## Usage

```
go run alice [input-file-path] [output-file-path]
```

## Usage as a library

Example program is main.go itself which can be referenced as
an example. 

## Configuration

Configuration (config.go) options will expand. For now, it only
monitors whether download progress should output to 
stdout, and configuration of tracker/DHT support.
