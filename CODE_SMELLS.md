# Code Smells Analysis

## High Priority Issues

### 1. **Unused Variable in terraformWithFilter** ⚠️
**Location:** [run/terraform_with_filter.go](run/terraform_with_filter.go#L71)

The `ignoreBlock` variable is declared but never set to `true`:
```go
ignoreBlock := false
// ... later ...
if ignoreBlock {
    goto NEXT
}
```

**Issue:** This code path is dead and suggests incomplete implementation.

**Fix:** Either implement block ignoring logic or remove the variable.

---

### 2. **Unused Import in terraform.go**
**Location:** [run/terraform.go](run/terraform.go#L4)

```go
import (
    "os"
)
```

The `os` package is imported but only `os.Stdin`, `os.Stdout`, `os.Stderr` are used, which come from stdin/stdout/stderr constants.

**Fix:** This import is actually needed for those constants. No action required.

---

### 3. **Inconsistent Error Handling for File Close**
**Location:** [run/terraform_with_filter.go](run/terraform_with_filter.go#L57) vs [run/terraform_with_progress.go](run/terraform_with_progress.go#L168)

**Bad Practice** (terraform_with_filter.go):
```go
defer func() { _ = file.Close() }()
```

**Good Practice** (terraform_with_progress.go):
```go
defer func() {
    if err := outputFile.Close(); err != nil {
        fmt.Fprintf(os.Stderr, "Error closing output file: %v\n", err)
    }
}()
```

**Issue:** Silently ignoring close errors can hide resource leaks or permission issues.

**Fix:** Use the pattern from terraform_with_progress.go consistently.

---

### 4. **Regex Compilation on Every Invocation**
**Location:** Multiple files

Files like [util/strings.go](util/strings.go), [util/file.go](util/file.go) compile regexes every time functions are called:

```go
func AddQuotes(input string) string {
    re := regexp.MustCompile(`\[([^"\]]*[A-Za-z_][^"\]]*)\]`)  // ← Recompiled every call
    return re.ReplaceAllStringFunc(input, func(match string) string {
        return `["` + re.FindStringSubmatch(match)[1] + `"]`
    })
}
```

**Issue:** Regex compilation is expensive; these should be compiled once at module initialization.

**Impact:** Performance degradation when processing large files.

**Fix:** Compile regexes at package level:
```go
var reAddQuotes = regexp.MustCompile(`\[([^"\]]*[A-Za-z_][^"\]]*)\]`)

func AddQuotes(input string) string {
    return reAddQuotes.ReplaceAllStringFunc(input, ...)
}
```

---

## Medium Priority Issues

### 5. **Duplicate Flag Handling Logic**
**Location:** [run/show.go](run/show.go#L17-L24)

```go
case "-no-output":
    noOutputs = true
case "-no-outputs":
    noOutputs = true
case "-no-output=false":
    noOutputs = false
case "-no-outputs=false":
    noOutputs = false
```

**Issue:** Duplicate cases for similar flags. Should normalize flag names.

**Fix:** Add a helper function to normalize flag names:
```go
case "-no-output", "-no-outputs":
    noOutputs = true
case "-no-output=false", "-no-outputs=false":
    noOutputs = false
```

---

### 6. **Global Variable with Side Effects**
**Location:** [run/exec_terraform_command.go](run/exec_terraform_command.go#L9)

```go
var TERRAFORM_PATH = os.Getenv("TERRAFORM_PATH")
```

**Issue:** 
- Evaluated at program initialization, doesn't reflect env var changes after startup
- UPPER_CASE naming suggests a constant, but it's a variable

**Fix:** Make it a function:
```go
func getTerraformPath() string {
    if path := os.Getenv("TERRAFORM_PATH"); path != "" {
        return path
    }
    // ... rest of logic
}
```

---

### 7. **No File Existence Check Before os.Stat**
**Location:** [run/show.go](run/show.go#L33)

```go
} else if _, err := os.Stat(arg); err == nil {
    newArgs = append(newArgs, arg)
} else {
    resources = append(resources, util.AddQuotes(arg))
}
```

**Issue:** Using `os.Stat` to guess whether something is a filename is unreliable (same issue as the original bug you fixed!). A path could exist as a file OR as a terraform resource name.

**Fix:** Remove this logic - just pass arguments directly to terraform:
```go
newArgs = append(newArgs, arg)
```

---

### 8. **Magic Constants**
**Location:** Several files use hardcoded values:

- Buffer size: `25*1024*1024` appears in multiple places
- Color codes: `\x1b[0m` hardcoded throughout
- Path patterns: `.terraform/providers`, `.terraform.lock.hcl` hardcoded in init.go

**Fix:** Define constants:
```go
const (
    BufferSize = 25 * 1024 * 1024
    ColorReset = "\x1b[0m"
    LockFile   = ".terraform.lock.hcl"
)
```

---

## Low Priority Issues

### 9. **Complex Function**
**Location:** [run/terraform_with_progress.go](run/terraform_with_progress.go) (~390 lines)

**Issue:** `terraformWithProgress()` is very long and handles multiple concerns:
- Argument parsing and validation
- Output filtering with complex regex patterns
- Progress tracking
- Color management
- File I/O

**Suggestion:** Consider breaking into smaller functions:
- `parseProgressArgs()` - extract argument parsing
- `compileRegexPatterns()` - centralize regex compilation
- `filterLine()` - line filtering logic

---

### 10. **No Constants for Regex Patterns**
**Location:** [run/init.go](run/init.go#L8-L32)

Pattern strings are defined inline making them:
- Hard to maintain
- Hard to reuse
- Not documented

**Fix:** Define patterns as named constants:
```go
const (
    patternInitOutput = `Finding .* versions matching|...`
    patternInitFooter = `(Terraform|OpenTofu).* has been successfully initialized!`
)
```

---

### 11. **Unused Variable in terraform.go**
**Location:** [run/init.go](run/init.go#L39)

```go
var codesign = false
```

While this is used, the pattern of declaring a var in a loop is unusual:
```go
var codesign = false

for _, arg := range args {
    switch util.ReplaceFirstTwoDashes(arg) {
    case "-codesign":
        codesign = true
```

Could be simplified with early return pattern.

---

## Summary Table

| Issue | Severity | Type | Count |
|-------|----------|------|-------|
| Unused variable (ignoreBlock) | High | Dead Code | 1 |
| Inconsistent error handling | High | Reliability | 1 |
| Regex recompilation | High | Performance | ~8 functions |
| File existence guessing | Medium | Correctness | 1 |
| Duplicate flag handling | Medium | Maintainability | 1 |
| Global initializer | Medium | Best Practices | 1 |
| Magic constants | Low | Maintainability | 3+ |
| Complex function | Low | Maintainability | 1 |
| Undocumented patterns | Low | Maintainability | 1 |

---

## Recommended Quick Wins

1. **Fix file close error handling** - 2 min, improves reliability
2. **Compile regexes at module level** - 15 min, improves performance
3. **Remove file existence checks** - 5 min, prevents bugs
4. **Define buffer size constant** - 5 min, improves maintainability
