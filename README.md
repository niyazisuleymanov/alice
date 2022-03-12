# alice

alice is a BitTorrent client written from scratch in Go.

## Usage

```
go build
./alice [input-file-path] [output-file-path]
```

## todo

- [ ] Sometimes tracker will not respond halting the whole peer request.
- [ ] Maybe use announce URLs in announce-list to get more peers?
- [ ] Too many handshakes fail. Is this a problem?
- [ ] Magnet link support
