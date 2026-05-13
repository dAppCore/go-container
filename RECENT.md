# Recent Changes

```
(HEAD)  feat(container): Apple provider full implementation — 17 tasks
        - Build, Encrypt, Decrypt real implementations
        - Stop, Kill, Remove, Logs, Exec, List, Inspect methods
        - Pull, Push, RemoveImage, ListImages methods
        - Version detection, Metal GPU passthrough, tracked map GC
        - StatusKilled, FormatOCI constants
        - capGPU wired on Apple Silicon via detectApple()

05f9e99 chore: sync go.mod dependencies
319ffe3 chore: add .core/ and .idea/ to .gitignore
d97537b fix: update stale import paths and dependency versions from extraction
6e786bb refactor: update import path from go-config to core/config
8910f74 docs: add CLAUDE.md project instructions
3ed0cf4 docs: add human-friendly documentation
a8e09bb feat: add cmd/vm from go-devops
8bc93ce feat: extract container/, devops/, sources/ from go-devops
68bac5d Initial commit
```
