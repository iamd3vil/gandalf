# Gandalf

Gandalf is a code generator for validating Go structs without runtime reflection. It reads `validate` struct tags and generates a `Validate() error` method containing ordinary Go comparisons.

```go
type Account struct {
	Username        string `validate:"required min:3 max:32"`
	Age             int    `validate:"min:18"`
	Password        string `validate:"required min:8"`
	ConfirmPassword string `validate:"eqfield:Password"`
	Email           string `validate:"required email"`
}
```

Running Gandalf generates code equivalent to:

```go
func (s *Account) Validate() error {
	if s.Username == "" {
		return errors.New("username can't be blank")
	}
	if len(s.Username) < 3 {
		return errors.New("username can't be less than 3")
	}
	// ...
	return nil
}
```

Validation therefore has no reflection overhead, and invalid values return the first failed constraint.

## Installation

Download a prebuilt binary from the [GitHub releases page](https://github.com/iamd3vil/gandalf/releases), or build Gandalf from source:

```sh
git clone https://github.com/iamd3vil/gandalf.git
cd gandalf
go install github.com/knadh/stuffbin/stuffbin@v1.1.0
make build
```

With Go 1.15, install stuffbin using `go get github.com/knadh/stuffbin/stuffbin` instead. The source build produces `gandalf.bin`; the module targets Go 1.15 or newer.

> Gandalf embeds its code-generation template into the executable with
> [stuffbin](https://github.com/knadh/stuffbin). A plain `go build` or
> `go install` does not perform this packing step; use `make build` when
> building from source.

## Usage

Add validation tags to structs in a package, then run:

```sh
gandalf -pkg account -dir ./account -file ./account/validations_gen.go
```

If built locally, use `./gandalf.bin` instead of `gandalf`.

The generated structs can then be validated directly:

```go
account := &Account{
	Username:        "gopher",
	Age:             21,
	Password:        "correct horse battery staple",
	ConfirmPassword: "correct horse battery staple",
	Email:           "gopher@example.com",
}

if err := account.Validate(); err != nil {
	log.Printf("invalid account: %v", err)
}
```

### Command-line options

| Flag | Default | Description |
| --- | --- | --- |
| `-pkg` | `main` | Package name to parse and use in the generated file. |
| `-dir` | `.` | Directory containing the package's Go source files. |
| `-file` | `<pkg>_validate_gen.go` | Path to the generated file. |

The package name passed to `-pkg` must match the package declaration in `-dir`.

Gandalf also works with `go generate`:

```go
//go:generate gandalf -pkg account -dir . -file validations_gen.go
```

Then regenerate with:

```sh
go generate ./...
```

## Constraints

Multiple constraints are separated by spaces and are evaluated from left to right.

| Constraint | Generated failure condition | Notes |
| --- | --- | --- |
| `required` | value is its zero value | Supports strings, booleans, every built-in numeric type (including complex types), slices, maps, and pointers. |
| `min:n` | value or length is less than `n` | Inclusive lower bound (`>= n`). Strings, slices, arrays, and maps use `len`. |
| `mineq:n` | value or length is less than or equal to `n` | Exclusive lower bound (`> n`). |
| `max:n` | value or length is greater than `n` | Inclusive upper bound (`<= n`). Strings and slices use `len`. |
| `maxeq:n` | value or length is greater than or equal to `n` | Exclusive upper bound (`< n`). |
| `eqfield:Field` | value differs from another field | Both fields must be comparable. |
| `regexp:pattern` | string does not match the pattern | The pattern is compiled once in generated package-level code. Single quotes around the pattern are removed. Patterns may contain colons; the pattern is validated with `regexp.Compile` and safely quoted during generation, so invalid patterns fail generation instead of panicking at package init. |
| `eq:value` | value is not equal to `value` | String values are treated as strings, numeric-looking values as numbers. |
| `ne:value` | value equals `value` | Inverse of `eq`. |
| `oneof:a,b,c` | value is not one of the listed values | Comma-separated list. Supports string fields. Wrap the entire value in single quotes if a value itself contains a comma (e.g. `oneof:'San Francisco, CA',Austin`). |
| `len:n` | length is not equal to `n` | For strings, slices, arrays, and maps. Uses `len`. |
| `unique` | slice contains duplicate values | For slices only. Element type must be comparable. |
| `contains:substr` | string does not contain `substr` | Uses `strings.Contains`. String fields only. |
| `excludes:substr` | string contains `substr` | Inverse of `contains`. String fields only. |
| `email` | string does not match email pattern | Uses `regexp.MustCompile` at call time. |
| `url` | string does not match URL pattern | Uses `regexp.MustCompile` at call time. |
| `uuid` | string does not match UUID pattern | Uses `regexp.MustCompile` at call time. |
| `ip` | string does not match IPv4 pattern | Uses `regexp.MustCompile` at call time. |
| `nefield:Field` | value equals `Field` | Both fields must be comparable. |
| `gtfield:Field` | value is not greater than `Field` | Numeric fields. |
| `ltfield:Field` | value is not less than `Field` | Numeric fields. |
| `gtefield:Field` | value is not greater than or equal to `Field` | Numeric fields. |
| `required_with:F1,F2` | named fields are all non-zero but this field is zero | Comma-separated list of other fields. For string fields. |
| `required_without:F1,F2` | named fields are all zero but this field is zero | Comma-separated list of other fields. For string fields. |
| `required_if:Field=value` | `Field` equals `value` but this field is zero | Format is `otherField=expectedValue`. |
| `validate` | calls `Validate()` on the nested field | For struct fields. Loses nested error detail (returns generic message). Supports value and pointer types. |

Examples:

```go
type Example struct {
	Name     string   `validate:"required min:3 max:50"`
	Score    float64  `validate:"mineq:0 maxeq:100"`
	Roles    []string `validate:"required min:1 unique"`
	Password string   `validate:"required"`
	Confirm  string   `validate:"eqfield:Password"`
	Code     string   `validate:"regexp:'^[A-Z]{3}-[0-9]{4}$'"`
	Country  string   `validate:"len:2"`
	Email    string   `validate:"email"`
	Website  string   `validate:"url"`
	Slug     string   `validate:"excludes:_"`
	Min      int      `validate:"required"`
	Max      int      `validate:"gtfield:Min"`
	Company  string   `validate:"required_if:Kind=business"`
	Profile  *Profile `validate:"validate"`
}
```

### Nested validation

The `validate` constraint delegates to the nested field's own generated (or
hand-written) `Validate() error` method, so a struct can be checked together
with the structs it embeds by value or by pointer:

```go
type Profile struct {
	Bio string `validate:"required max:200"`
}

type Account struct {
	Profile  Profile  `validate:"validate"`
	Settings *Profile `validate:"validate"`
}
```

The generated code calls `Validate()` on `s.Profile`, and for pointer fields
only after a `nil` check, so a `nil` pointer is skipped rather than
dereferenced. The nested type must itself have a `Validate() error` method —
either generated by Gandalf in the same run or written by hand.

Current limitation: the nested error detail is lost. Because every constraint
renders as a boolean condition followed by `errors.New(...)`, a failing nested
field reports a generic message (for example `profile validation failed`)
instead of wrapping the underlying error. Call the nested `Validate()` directly
if you need the specific inner cause.

## Generated code

Generated files:

- contain a `// Code generated by gandalf ... DO NOT EDIT.` header;
- define `Validate() error` on a pointer receiver for every struct with validation tags, and always in a deterministic, sorted-by-struct-name order so repeated generation from the same input produces byte-identical output;
- return the first validation error;
- precompile regular expressions as package-level variables, declared in the same deterministic struct/field order; and
- use only standard-library packages at runtime.

Commit generated files if consumers of your package should not need Gandalf installed, or regenerate them as part of your build process.

## Current limitations

Gandalf is an early-stage generator and intentionally rejects combinations it cannot generate safely:

- `required` is not supported for arrays, direct qualified values, or user-defined named value types; use a supported length constraint or a pointer where appropriate;
- `min` and `max` variants are supported only for built-in numeric values and types with a usable length (strings, slices, arrays, and maps);
- tagged embedded fields and field expressions such as functions, channels, inline structs, and interfaces are rejected; untagged embedded fields are ignored; and
- regular expressions apply only to strings and cannot contain spaces because constraints are separated with whitespace;
- constraint values cannot contain spaces, since constraints are separated with whitespace — this affects `regexp` patterns, `contains`/`excludes` substrings, and any other single-value constraint; multi-value constraints (`oneof`, `required_with`, `required_without`) use commas and are not affected; and
- `validate` reports a generic message rather than wrapping the nested error (see [Nested validation](#nested-validation)).

Malformed tags, unknown constraints, invalid regular expressions, unsupported field types, and unsupported constraint/type combinations produce generation errors with struct and field context.

Review generated code before using it in production.

## Development

Build and run the test suite with:

```sh
make test
```

This rebuilds Gandalf, regenerates `test/validations_gen.go`, and runs the tests in `./test`.

## License

Gandalf is released under the [MIT License](LICENSE).
