# Plain HTML Converter

A Go CLI tool that converts HTML files to Plain Go code, with intelligent detection of full pages vs snippets, and support for htmx and Alpine.js attributes.

## Installation

```bash
# Clone and build
git clone https://github.com/plainkit/converter
cd converter
go build -o plainkit-converter

# Or install directly
go install github.com/plainkit/converter@latest
```

## Usage

### Basic Usage

```bash
# Convert HTML from stdin
echo '<div class="container">Hello</div>' | plainkit-converter

# Convert HTML file
plainkit-converter examples/basic.html

# Save to file
plainkit-converter examples/basic.html -o component.go
```

### With HTMX Support

```bash
# Enable htmx attribute conversion
plainkit-converter --htmx examples/htmx.html
```

### With Alpine.js Support

```bash
# Enable Alpine.js attribute conversion
plainkit-converter --alpine examples/alpine.html
```

### Combined Support

```bash
# Enable both htmx and Alpine.js
plainkit-converter --htmx --alpine examples/combined.html
```

## Examples

### Full HTML Page

Input:
```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>My Page</title>
</head>
<body>
    <div class="container">
        <h1>Welcome</h1>
        <p>Hello world!</p>
    </div>
</body>
</html>
```

Output:
```go
package main

import (
    . "github.com/plainkit/html"
)

func Page() Node {
    return Html(
        Head(
            Meta(Charset("utf-8")),
            Title(Text("My Page"))
        ),
        Body(
            Div(
                Class("container"),
                H1(Text("Welcome")),
                P(Text("Hello world!"))
            )
        )
    )
}
```

### HTML Snippet

Input:
```html
<div class="container">
    <h1>Welcome</h1>
    <p>Hello <strong>world</strong>!</p>
</div>
```

Output:
```go
package main

import (
    . "github.com/plainkit/html"
)

func Component() Node {
    return Div(
        Class("container"),
        H1(Text("Welcome")),
        P(
            Text("Hello"),
            Strong(Text("world")),
            Text("!")
        )
    )
}
```

### Multiple Fragments

Input:
```html
<div>First</div><p>Second</p><span>Third</span>
```

Output:
```go
package main

import (
    . "github.com/plainkit/html"
)

func Components() []Node {
    return []Node{
        Div(Text("First")),
        P(Text("Second")),
        Span(Text("Third"))
    }
}
```

### HTMX Example

Input:
```html
<button hx-get="/api/data" hx-target="#result" hx-trigger="click">
    Load Data
</button>
<div id="result"></div>
```

Output:
```go
package main

import (
    . "github.com/plainkit/html"
    "github.com/plainkit/htmx"
)

func Component() Node {
    return Button(
        htmx.HxGet("/api/data"),
        htmx.HxTarget("#result"),
        htmx.HxTrigger("click"),
        Text("Load Data")
    ),
    Div(
        Id("result")
    )
}
```

### Alpine.js Example

Input:
```html
<div x-data="{ count: 0 }">
    <button @click="count++">+</button>
    <span x-text="count"></span>
    <button @click="count--">-</button>
</div>
```

Output:
```go
package main

import (
    . "github.com/plainkit/html"
    "github.com/plainkit/alpine"
)

func Component() Node {
    return Div(
        alpine.XData("{ count: 0 }"),
        Button(
            alpine.AtClick("count++"),
            Text("+")
        ),
        Span(
            alpine.XText("count")
        ),
        Button(
            alpine.AtClick("count--"),
            Text("-")
        )
    )
}
```

## Supported Features

### Standard HTML Attributes
- Class, ID, style attributes
- Form attributes (action, method, type, name, value, etc.)
- Link attributes (href, rel, target)
- Image attributes (src, alt, width, height)
- Boolean attributes (disabled, checked, required, etc.)
- Data and ARIA attributes
- Meta tag attributes

### HTMX Attributes (with --htmx flag)
- HTTP methods: `hx-get`, `hx-post`, `hx-put`, `hx-delete`, `hx-patch`
- Targeting: `hx-target`, `hx-swap`, `hx-swap-oob`
- Triggers: `hx-trigger`, `hx-indicator`
- Request config: `hx-headers`, `hx-vals`, `hx-params`
- Navigation: `hx-boost`, `hx-push-url`, `hx-replace-url`
- User interaction: `hx-confirm`, `hx-prompt`
- Advanced: `hx-ext`, `hx-sse`, `hx-ws`, `hx-sync`

### Alpine.js Attributes (with --alpine flag)
- Core directives: `x-data`, `x-init`, `x-show`, `x-if`, `x-for`
- Content: `x-html`, `x-text`, `x-cloak`
- Events: `@click`, `@submit`, `@change`, etc. (including modifiers)
- Binding: `:class`, `:style`, `:disabled`, etc.
- Forms: `x-model` (including modifiers like `.lazy`, `.number`)
- Advanced: `x-transition`, `x-effect`, `x-ref`, `x-teleport`

## Command Line Options

```
Usage:
  plainkit-converter [input] [flags]

Flags:
      --alpine    Enable Alpine.js attribute conversion
      --htmx      Enable htmx attribute conversion
  -o, --output    Output file (default: stdout)
  -v, --version   Show version
  -h, --help      Help for plainkit-converter
```

## Testing

```bash
# Run tests
go test -v

# Test with example files
plainkit-converter examples/basic.html
plainkit-converter --htmx examples/htmx.html
plainkit-converter --alpine examples/alpine.html
plainkit-converter --htmx --alpine examples/combined.html
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new features
4. Submit a pull request

## License

MIT License - see LICENSE file for details.
