## Custom MCPs servers for different purposes

This repo contains multiple custome MCPs servers in Go that I built for solving different needs I had.
Feel free to use and modify them as you need, you can contribute to this project suggesting features, fixes, improvements and so.

### Structure

```
./ -
    |- README.md - This file
    |- framework - Shared code between servers
    |- cmd/server
       |- cloudflare
```

Each `cmd/server/*` is and independent application that implements the framework to start an MCP server, each of them has its own dockerfile and can be runned from a docker container or equivalent.
Each of them should have its own README.md describing what each server does and how it works
