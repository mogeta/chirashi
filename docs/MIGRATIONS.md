# Migration Notes

This file tracks migration steps for breaking changes in API or config behavior.

## Unreleased

### Import path guidance (canonical path update)

Recommended import path is now:

```go
import "github.com/mogeta/chirashi"
```

Compatibility path remains available:

```go
import "github.com/mogeta/chirashi/particle"
```

Action:
- Prefer `github.com/mogeta/chirashi` in new code.
- Existing projects may keep `.../particle` temporarily and migrate when convenient.

