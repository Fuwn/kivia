# &#127984; Kivia

> Self-documenting Name Analyser for Go

Name things so your code explains itself.

Kivia is a fast, local-only analyser that flags identifiers whose terms are absent from dictionary sources or resemble abbreviations of dictionary words. It is built for teams that want explicit, readable naming conventions without external AI calls.

## Philosophy

Identifier names should be fully self-documenting.

Kivia enforces a strict readability standard:

- Prefer full words over shorthand
- Avoid ambiguous abbreviations
- Keep the naming intent clear from the identifier itself

Examples:

- `userNum` → invalid (`num` is an abbreviation)
- `ctx` → invalid (`ctx` is an abbreviation)
- `userCount` → valid
- `requestContext` → valid

## Rules

1. **Dictionary words pass**: If a token is present in the loaded dictionary sources, it passes.
2. **Abbreviations are violations**: If a token expands to a dictionary word (for example, `ctx` -> `context`), it is flagged.
3. **Unknown terms are violations**: If a token is not in the dictionary and does not map to a known expansion, it is flagged.
4. **Minimum length is explicit**: `--min-eval-length` determines whether short identifiers are evaluated.

Kivia also applies dictionary-backed spelling-variant matching for common British/American pairs (for example `normalise`/`normalize`, `colour`/`color`, `centre`/`center`).

## How It Works

Kivia parses Go source using the standard library's AST, extracts identifiers, tokenises names (camel, snake, or kebab), and evaluates each token against a local NLP dictionary pipeline.

- No network requests
- No LLM/API dependency
- Deterministic local analysis

## Installation

```bash
go install github.com/Fuwn/kivia@latest
```

Or build locally:

```bash
go build -o ./bin/kivia .
```

## Usage

```bash
# Analyse a package tree
kivia --path ./...

# Ignore single-letter names during evaluation
kivia --path ./... --min-eval-length 2

# Ignore specific violations
kivia --path ./... --ignore name=ctx --ignore file=testdata

# JSON output without context payload
kivia --path ./... --format json --omit-context
```

### Flags

| Flag | Description |
|------|-------------|
| `--path` | Path to analyse (`directory`, `file`, or `./...`). |
| `--omit-context` | Hide usage context in output. |
| `--min-eval-length` | Minimum identifier length in runes to evaluate (must be `>= 1`). |
| `--format` | Output format: `text` or `json`. |
| `--fail-on-violation` | Exit with code `1` when violations are found. |
| `--ignore` | Ignore violations by matcher. Repeatable. Prefixes: `name=`, `kind=`, `file=`, `reason=`, `func=`. |

## Ignore Matchers

`--ignore` supports targeted filtering:

- `name=<substring>`
- `kind=<substring>`
- `file=<substring>`
- `reason=<substring>`
- `func=<substring>`

Without a prefix, the matcher is applied as a substring across all violation fields.

Example:

```bash
kivia --path ./... \
  --ignore name=ctx \
  --ignore reason=abbreviation \
  --ignore file=_test.go
```

## Output

### Text (default)

```text
internal/example/sample.go:12:9 parameter "ctx": Contains abbreviation: ctx.
  context: type=context.Context, function=Handle
```

### JSON

```json
{
  "violations": [
    {
      "identifier": {
        "name": "ctx",
        "kind": "parameter",
        "file": "internal/example/sample.go",
        "line": 12,
        "column": 9,
        "context": {
          "enclosingFunction": "Handle",
          "type": "context.Context"
        }
      },
      "reason": "Contains abbreviation: ctx."
    }
  ]
}
```

## Identifier Scope (Go)

Kivia currently extracts and evaluates:

- Types
- Functions and methods
- Receivers
- Parameters
- Named results
- Variables (`var`/`const` and `:=`)
- Range keys and values
- Struct fields
- Interface methods

## Dictionary and NLP Source

Kivia loads dictionary data only from configured/system dictionary files.

1. `KIVIA_DICTIONARY_PATH` (optional): one path or multiple paths separated by your OS path separator (`:` on macOS/Linux, `;` on Windows). Commas are also accepted.
2. If `KIVIA_DICTIONARY_PATH` is not set, Kivia uses a default set of dictionary files (for example, `/usr/share/dict/words`, `/usr/share/dict/web2`, and Hunspell dictionaries when present).
3. If no usable words are found, the analysis fails with an error.

## License

Licensed under either of [Apache License, Version 2.0](LICENSE-APACHE) or
[MIT license](LICENSE-MIT) at your option.

Unless you explicitly state otherwise, any contribution intentionally submitted
for inclusion in this crate by you, as defined in the Apache-2.0 license, shall
be dual licensed as above, without any additional terms or conditions.
