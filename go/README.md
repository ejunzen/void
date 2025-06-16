# Go CLI for Void

This directory contains a small Go-based command line tool demonstrating how the
`directoryStrService`, answering questions, and calling the MCP tool could be
reimplemented in Go.

## Building

```
go build ./cmd/voidcli
```

## Usage

```
voidcli -action=index -dir /path/to/dir -depth 2
voidcli -action=ask -question "Explain this project" # requires OPENAI_API_KEY
voidcli -action=mcp -server myserver -tool greet -params '{"name":"bob"}'
```

The MCP servers are read from `~/.void/mcp.json` by default, which should match
Void's configuration format.
