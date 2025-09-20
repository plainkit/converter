package main

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Converter handles HTML to Plain conversion
type Converter struct {
	useHTMX   bool
	useAlpine bool
	imports   map[string]bool
	indent    int
}

// NewConverter creates a new HTML to Plain converter
func NewConverter(useHTMX, useAlpine bool) *Converter {
	return &Converter{
		useHTMX:   useHTMX,
		useAlpine: useAlpine,
		imports:   make(map[string]bool),
		indent:    0,
	}
}

// Convert converts HTML string to Plain Go code
func (c *Converter) Convert(htmlContent string) (string, error) {
	// Clean up the content
	htmlContent = strings.TrimSpace(htmlContent)

	// Check if this looks like a full HTML document
	isFullPage := strings.Contains(htmlContent, "<!DOCTYPE") ||
		strings.Contains(htmlContent, "<html") ||
		(strings.Contains(htmlContent, "<head") && strings.Contains(htmlContent, "<body"))

	if isFullPage {
		return c.convertFullPage(htmlContent)
	}

	// Handle as snippet/fragment
	return c.convertFragment(htmlContent)
}

// convertFullPage handles complete HTML documents
func (c *Converter) convertFullPage(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find the html element
	var htmlNode *html.Node
	var findHTML func(*html.Node)
	findHTML = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "html" {
			htmlNode = n
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			findHTML(child)
		}
	}
	findHTML(doc)

	if htmlNode == nil {
		return "", fmt.Errorf("no html element found")
	}

	var buf bytes.Buffer
	c.collectImports(htmlNode)
	buf.WriteString(c.generateImports())
	buf.WriteString("\n")
	buf.WriteString("func Page() Node {\n")
	buf.WriteString("\treturn ")
	code := c.convertNode(htmlNode, 1)
	buf.WriteString(code)
	buf.WriteString("\n}\n")
	return buf.String(), nil
}

// convertFragment handles HTML snippets/fragments
func (c *Converter) convertFragment(htmlContent string) (string, error) {
	// First try to parse as fragment
	fragments, err := html.ParseFragment(strings.NewReader(htmlContent), nil)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML fragment: %w", err)
	}

	if len(fragments) == 0 {
		return "", fmt.Errorf("no fragments found")
	}

	// Extract actual content from the parsed fragments
	// ParseFragment may wrap content in html/head/body, so we need to unwrap it
	var actualContent []*html.Node
	for _, frag := range fragments {
		extracted := c.extractActualContent(frag)
		actualContent = append(actualContent, extracted...)
	}

	// Filter out whitespace-only text nodes
	var validFragments []*html.Node
	for _, frag := range actualContent {
		if frag.Type == html.TextNode && strings.TrimSpace(frag.Data) == "" {
			continue
		}
		validFragments = append(validFragments, frag)
	}

	if len(validFragments) == 0 {
		return "", fmt.Errorf("no convertible content found")
	}

	var buf bytes.Buffer
	c.collectImportsFromFragments(validFragments)
	buf.WriteString(c.generateImports())
	buf.WriteString("\n")

	if len(validFragments) == 1 {
		// Single fragment - return it directly
		buf.WriteString("func Component() Node {\n")
		buf.WriteString("\treturn ")
		code := c.convertNode(validFragments[0], 1)
		buf.WriteString(code)
		buf.WriteString("\n}\n")
	} else {
		// Multiple fragments - return as slice
		buf.WriteString("func Components() []Node {\n")
		buf.WriteString("\treturn []Node{\n")
		for _, frag := range validFragments {
			buf.WriteString("\t\t")
			code := c.convertNode(frag, 2)
			buf.WriteString(code)
			buf.WriteString(",")
			buf.WriteString("\n")
		}
		buf.WriteString("\t}\n}\n")
	}
	return buf.String(), nil
}

// extractActualContent recursively extracts the meaningful content from parsed fragments
func (c *Converter) extractActualContent(n *html.Node) []*html.Node {
	var result []*html.Node

	if n.Type == html.ElementNode {
		switch n.Data {
		case "html", "head", "body":
			// Skip wrapper elements, extract their children
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				result = append(result, c.extractActualContent(child)...)
			}
		default:
			// This is actual content
			result = append(result, n)
		}
	} else if n.Type == html.TextNode && strings.TrimSpace(n.Data) != "" {
		// Non-empty text node
		result = append(result, n)
	}

	return result
}

// collectImportsFromFragments collects imports from multiple fragments
func (c *Converter) collectImportsFromFragments(fragments []*html.Node) {
	c.imports["github.com/plainkit/html"] = true

	for _, frag := range fragments {
		c.collectImports(frag)
	}
}

// collectImports walks the tree to determine needed imports
func (c *Converter) collectImports(n *html.Node) {
	c.imports["github.com/plainkit/html"] = true

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			for _, attr := range node.Attr {
				if strings.HasPrefix(attr.Key, "hx-") && c.useHTMX {
					c.imports["github.com/plainkit/htmx"] = true
				}
				if (strings.HasPrefix(attr.Key, "x-") ||
					strings.HasPrefix(attr.Key, "@") ||
					strings.HasPrefix(attr.Key, ":")) && c.useAlpine {
					c.imports["github.com/plainkit/alpine"] = true
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
}

// generateImports generates the import statements
func (c *Converter) generateImports() string {
	var buf bytes.Buffer
	buf.WriteString("package main\n\n")
	buf.WriteString("import (\n")

	// Always import html with dot import for convenience
	buf.WriteString("\t. \"github.com/plainkit/html\"\n")

	if c.imports["github.com/plainkit/htmx"] {
		buf.WriteString("\t\"github.com/plainkit/htmx\"\n")
	}
	if c.imports["github.com/plainkit/alpine"] {
		buf.WriteString("\t\"github.com/plainkit/alpine\"\n")
	}

	buf.WriteString(")\n")
	return buf.String()
}

// convertNode converts an HTML node to Plain code
func (c *Converter) convertNode(n *html.Node, depth int) string {
	switch n.Type {
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text == "" {
			return ""
		}
		return fmt.Sprintf("T(%s)", c.quoteValue(text))

	case html.ElementNode:
		return c.convertElement(n, depth)

	case html.DocumentNode:
		// Process children
		var children []string
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if code := c.convertNode(child, depth); code != "" {
				children = append(children, code)
			}
		}
		if len(children) == 1 {
			return children[0]
		}
		return strings.Join(children, ",\n"+strings.Repeat("\t", depth))

	default:
		return ""
	}
}

// convertElement converts an HTML element to Plain code
func (c *Converter) convertElement(n *html.Node, depth int) string {
	var buf bytes.Buffer

	// Convert tag name to Plain function with context
	funcName := c.tagToFunctionWithContext(n.Data, n)
	buf.WriteString(funcName)
	buf.WriteString("(")

	var args []string

	// Process attributes
	for _, attr := range n.Attr {
		if attrCode := c.convertAttribute(attr, n.Data); attrCode != "" {
			args = append(args, attrCode)
		}
	}

	// Process children
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if code := c.convertNode(child, depth+1); code != "" {
			args = append(args, code)
		}
	}

	if len(args) > 0 {
		if len(args) > 3 || containsMultilineContent(args) {
			// Multi-line formatting
			buf.WriteString("\n")
			for _, arg := range args {
				buf.WriteString(strings.Repeat("\t", depth+1))
				buf.WriteString(arg)
				buf.WriteString(",")
				buf.WriteString("\n")
			}
			buf.WriteString(strings.Repeat("\t", depth))
		} else {
			// Single line
			buf.WriteString(strings.Join(args, ", "))
		}
	}

	buf.WriteString(")")
	return buf.String()
}

// tagToFunctionWithContext converts HTML tag names to Plain function names with context awareness
func (c *Converter) tagToFunctionWithContext(tag string, node *html.Node) string {
	// Special cases with context-specific function names
	specialTags := map[string]string{
		"a":          "A",
		"b":          "B",
		"i":          "I",
		"p":          "P",
		"br":         "Br",
		"hr":         "Hr",
		"h1":         "H1",
		"h2":         "H2",
		"h3":         "H3",
		"h4":         "H4",
		"h5":         "H5",
		"h6":         "H6",
		"ul":         "Ul",
		"ol":         "Ol",
		"li":         "Li",
		"dl":         "Dl",
		"dt":         "Dt",
		"dd":         "Dd",
		"em":         "Em",
		"abbr":       "Abbr",
		"kbd":        "Kbd",
		"var":        "Var",
		"dfn":        "Dfn",
		"del":        "Del",
		"ins":        "Ins",
		"sub":        "Sub",
		"sup":        "Sup",
		"col":        "Col",
		"colgroup":   "ColGroup",
		"tbody":      "Tbody",
		"thead":      "Thead",
		"tfoot":      "Tfoot",
		"tr":         "Tr",
		"td":         "Td",
		"th":         "Th",
		"fieldset":   "Fieldset",
		"legend":     "Legend",
		"datalist":   "Datalist",
		"optgroup":   "OptGroup",
		"textarea":   "Textarea",
		"blockquote": "Blockquote",
		"figcaption": "Figcaption",
	}

	if special, ok := specialTags[tag]; ok {
		return special
	}

	// Check for context-specific function names
	switch tag {
	case "title":
		// Check if we're in a head context by looking at ancestors
		if c.isInHeadContext(node) {
			return "HeadTitle"
		}
		return "Title"
	case "label":
		// Check if we're in a form context
		if c.isInFormContext(node) {
			return "FormLabel"
		}
		return "Label"
	}

	// Title case conversion for standard tags
	caser := cases.Title(language.English)
	return caser.String(tag)
}

// isInHeadContext checks if a node is within a head element
func (c *Converter) isInHeadContext(node *html.Node) bool {
	current := node.Parent
	for current != nil {
		if current.Type == html.ElementNode && current.Data == "head" {
			return true
		}
		current = current.Parent
	}
	return false
}

// isInFormContext checks if a node is within a form element
func (c *Converter) isInFormContext(node *html.Node) bool {
	current := node.Parent
	for current != nil {
		if current.Type == html.ElementNode && current.Data == "form" {
			return true
		}
		current = current.Parent
	}
	return false
}

// quoteValue properly quotes a string value, using backticks for multiline content
func (c *Converter) quoteValue(val string) string {
	// Check if the value contains newlines or is complex JavaScript
	if strings.Contains(val, "\n") || (len(val) > 50 && (strings.Contains(val, "{") || strings.Contains(val, "function"))) {
		// Use backticks for multiline or complex content
		// Escape any existing backticks
		val = strings.ReplaceAll(val, "`", "` + \"`\" + `")
		return fmt.Sprintf("`%s`", val)
	}

	// Use regular double quotes for simple content
	val = strings.ReplaceAll(val, `"`, `\"`)
	return fmt.Sprintf("\"%s\"", val)
}

// convertAttribute converts HTML attributes to Plain attributes
func (c *Converter) convertAttribute(attr html.Attribute, tagName string) string {
	key := attr.Key
	val := attr.Val

	// Handle htmx attributes
	if strings.HasPrefix(key, "hx-") && c.useHTMX {
		return c.convertHTMXAttribute(key, val)
	}

	// Handle Alpine.js attributes
	if c.useAlpine {
		if strings.HasPrefix(key, "x-") {
			return c.convertAlpineAttribute(key, val)
		}
		if strings.HasPrefix(key, "@") {
			return c.convertAlpineEventAttribute(key, val)
		}
		if strings.HasPrefix(key, ":") {
			return c.convertAlpineBindAttribute(key, val)
		}
	}

	// Handle standard HTML attributes with context-specific functions
	switch key {
	case "class":
		return fmt.Sprintf("Class(%s)", c.quoteValue(val))
	case "id":
		return fmt.Sprintf("Id(%s)", c.quoteValue(val))
	case "style":
		return fmt.Sprintf("Style(%s)", c.quoteValue(val))
	case "href":
		return fmt.Sprintf("Href(%s)", c.quoteValue(val))
	case "src":
		if tagName == "script" {
			return fmt.Sprintf("ScriptSrc(%s)", c.quoteValue(val))
		}
		return fmt.Sprintf("Src(%s)", c.quoteValue(val))
	case "type":
		if tagName == "input" {
			return fmt.Sprintf("InputType(%s)", c.quoteValue(val))
		}
		if tagName == "button" {
			return fmt.Sprintf("ButtonType(%s)", c.quoteValue(val))
		}
		return fmt.Sprintf("Type(%s)", c.quoteValue(val))
	case "value":
		if tagName == "input" {
			return fmt.Sprintf("InputValue(%s)", c.quoteValue(val))
		}
		return fmt.Sprintf("Value(%s)", c.quoteValue(val))
	case "name":
		if tagName == "input" {
			return fmt.Sprintf("InputName(%s)", c.quoteValue(val))
		}
		if tagName == "meta" {
			return fmt.Sprintf("Name(%s)", c.quoteValue(val))
		}
		return fmt.Sprintf("Name(%s)", c.quoteValue(val))
	case "placeholder":
		return fmt.Sprintf("Placeholder(%s)", c.quoteValue(val))
	case "disabled":
		return "Disabled()"
	case "checked":
		return "Checked()"
	case "readonly":
		return "ReadOnly()"
	case "required":
		return "Required()"
	case "multiple":
		return "Multiple()"
	case "selected":
		return "Selected()"
	case "defer":
		return "Defer()"
	case "async":
		return "Async()"
	case "charset":
		return fmt.Sprintf("Charset(%s)", c.quoteValue(val))
	case "content":
		return fmt.Sprintf("Content(%s)", c.quoteValue(val))
	case "method":
		return fmt.Sprintf("Method(%s)", c.quoteValue(val))
	case "action":
		return fmt.Sprintf("Action(%s)", c.quoteValue(val))
	case "target":
		return fmt.Sprintf("Target(%s)", c.quoteValue(val))
	case "rel":
		return fmt.Sprintf("Rel(%s)", c.quoteValue(val))
	case "alt":
		return fmt.Sprintf("Alt(%s)", c.quoteValue(val))
	case "title":
		return fmt.Sprintf("Title(%s)", c.quoteValue(val))
	case "width":
		return fmt.Sprintf("Width(%s)", c.quoteValue(val))
	case "height":
		return fmt.Sprintf("Height(%s)", c.quoteValue(val))
	case "colspan":
		return fmt.Sprintf("ColSpan(%s)", c.quoteValue(val))
	case "rowspan":
		return fmt.Sprintf("RowSpan(%s)", c.quoteValue(val))
	case "for":
		return fmt.Sprintf("For(%s)", c.quoteValue(val))
	case "maxlength":
		return fmt.Sprintf("MaxLength(%s)", c.quoteValue(val))
	case "minlength":
		return fmt.Sprintf("MinLength(%s)", c.quoteValue(val))
	case "min":
		return fmt.Sprintf("Min(%s)", c.quoteValue(val))
	case "max":
		return fmt.Sprintf("Max(%s)", c.quoteValue(val))
	case "step":
		return fmt.Sprintf("Step(%s)", c.quoteValue(val))
	case "pattern":
		return fmt.Sprintf("Pattern(%s)", c.quoteValue(val))
	case "rows":
		return fmt.Sprintf("Rows(%s)", c.quoteValue(val))
	case "cols":
		return fmt.Sprintf("Cols(%s)", c.quoteValue(val))
	case "autocomplete":
		return fmt.Sprintf("AutoComplete(%s)", c.quoteValue(val))
	case "autofocus":
		return "Autofocus()"
	default:
		// Handle data- and aria- attributes
		if strings.HasPrefix(key, "data-") {
			dataKey := strings.TrimPrefix(key, "data-")
			return fmt.Sprintf("Data(%s, %s)", c.quoteValue(dataKey), c.quoteValue(val))
		}
		if strings.HasPrefix(key, "aria-") {
			ariaKey := strings.TrimPrefix(key, "aria-")
			return fmt.Sprintf("Aria(%s, %s)", c.quoteValue(ariaKey), c.quoteValue(val))
		}
		if key == "role" {
			return fmt.Sprintf("Role(%s)", c.quoteValue(val))
		}
		if key == "tabindex" {
			return fmt.Sprintf("TabIndex(%s)", c.quoteValue(val))
		}
		// For any unknown attributes, use Custom
		return fmt.Sprintf("Custom(%s, %s)", c.quoteValue(key), c.quoteValue(val))
	}
}

// convertHTMXAttribute converts htmx attributes
func (c *Converter) convertHTMXAttribute(key, val string) string {
	// Map hx- attributes to htmx functions
	htmxMap := map[string]string{
		"hx-get":          "HxGet",
		"hx-post":         "HxPost",
		"hx-put":          "HxPut",
		"hx-patch":        "HxPatch",
		"hx-delete":       "HxDelete",
		"hx-trigger":      "HxTrigger",
		"hx-target":       "HxTarget",
		"hx-swap":         "HxSwap",
		"hx-swap-oob":     "HxSwapOob",
		"hx-indicator":    "HxIndicator",
		"hx-push-url":     "HxPushUrl",
		"hx-replace-url":  "HxReplaceUrl",
		"hx-select":       "HxSelect",
		"hx-select-oob":   "HxSelectOob",
		"hx-vals":         "HxVals",
		"hx-headers":      "HxHeaders",
		"hx-include":      "HxInclude",
		"hx-params":       "HxParams",
		"hx-confirm":      "HxConfirm",
		"hx-prompt":       "HxPrompt",
		"hx-validate":     "HxValidate",
		"hx-disabled-elt": "HxDisabledElt",
		"hx-ext":          "HxExt",
		"hx-boost":        "HxBoost",
		"hx-preserve":     "HxPreserve",
		"hx-sse":          "HxSse",
		"hx-ws":           "HxWs",
		"hx-sync":         "HxSync",
		"hx-encoding":     "HxEncoding",
		"hx-disinherit":   "HxDisinherit",
	}

	if funcName, ok := htmxMap[key]; ok {
		if key == "hx-boost" || key == "hx-preserve" || key == "hx-validate" {
			// Boolean attributes
			if val == "true" {
				return fmt.Sprintf("htmx.%s()", funcName)
			}
			return fmt.Sprintf("htmx.%s(%v)", funcName, val == "true")
		}
		return fmt.Sprintf("htmx.%s(%s)", funcName, c.quoteValue(val))
	}

	// Fallback for any unknown hx- attributes
	return fmt.Sprintf("Custom(%s, %s)", c.quoteValue(key), c.quoteValue(val))
}

// convertAlpineAttribute converts Alpine.js x- attributes
func (c *Converter) convertAlpineAttribute(key, val string) string {
	// Map x- attributes to alpine functions
	alpineMap := map[string]string{
		"x-data":                   "XData",
		"x-init":                   "XInit",
		"x-show":                   "XShow",
		"x-if":                     "XIf",
		"x-for":                    "XFor",
		"x-html":                   "XHtml",
		"x-text":                   "XText",
		"x-model":                  "XModel",
		"x-modelable":              "XModelable",
		"x-effect":                 "XEffect",
		"x-ref":                    "XRef",
		"x-teleport":               "XTeleport",
		"x-ignore":                 "XIgnore",
		"x-id":                     "XId",
		"x-cloak":                  "XCloak",
		"x-transition":             "XTransition",
		"x-transition:enter":       "XTransitionEnter",
		"x-transition:enter-start": "XTransitionEnterStart",
		"x-transition:enter-end":   "XTransitionEnterEnd",
		"x-transition:leave":       "XTransitionLeave",
		"x-transition:leave-start": "XTransitionLeaveStart",
		"x-transition:leave-end":   "XTransitionLeaveEnd",
		"x-model.lazy":             "XModelLazy",
		"x-model.number":           "XModelNumber",
	}

	// Check for x-on:event format
	if strings.HasPrefix(key, "x-on:") {
		event := strings.TrimPrefix(key, "x-on:")
		return fmt.Sprintf("alpine.XOn(%s, %s)", c.quoteValue(event), c.quoteValue(val))
	}

	// Check for x-bind:attr format
	if strings.HasPrefix(key, "x-bind:") {
		attr := strings.TrimPrefix(key, "x-bind:")
		return fmt.Sprintf("alpine.XBind(%s, %s)", c.quoteValue(attr), c.quoteValue(val))
	}

	// Check for x-model with debounce
	if strings.HasPrefix(key, "x-model.debounce") {
		parts := strings.Split(key, ".")
		if len(parts) > 2 {
			delay := parts[2]
			return fmt.Sprintf("alpine.XModelDebounce(%s, %s)", c.quoteValue(val), c.quoteValue(delay))
		}
	}

	if funcName, ok := alpineMap[key]; ok {
		if key == "x-cloak" || key == "x-ignore" || key == "x-transition" {
			// No-argument attributes
			return fmt.Sprintf("alpine.%s()", funcName)
		}
		return fmt.Sprintf("alpine.%s(%s)", funcName, c.quoteValue(val))
	}

	// Fallback for any unknown x- attributes
	return fmt.Sprintf("Custom(%s, %s)", c.quoteValue(key), c.quoteValue(val))
}

// convertAlpineEventAttribute converts Alpine @ event attributes
func (c *Converter) convertAlpineEventAttribute(key, val string) string {
	// Remove @ prefix
	eventPart := strings.TrimPrefix(key, "@")

	// Check for modifiers
	parts := strings.Split(eventPart, ".")
	event := parts[0]

	if len(parts) > 1 {
		// Has modifiers
		modifiers := strings.Join(parts[1:], ".")

		// Common event+modifier combinations
		commonCombos := map[string]string{
			"click.away":     "AtClickAway",
			"click.outside":  "AtClickOutside",
			"click.prevent":  "AtClickPrevent",
			"click.stop":     "AtClickStop",
			"submit.prevent": "AtSubmitPrevent",
			"keydown.escape": "AtKeydownEscape",
			"keydown.enter":  "AtKeydownEnter",
			"keydown.window": "AtKeydownWindow",
		}

		combo := event + "." + modifiers
		if funcName, ok := commonCombos[combo]; ok {
			return fmt.Sprintf("alpine.%s(%s)", funcName, c.quoteValue(val))
		}

		// Generic @ with modifiers
		return fmt.Sprintf("Custom(%s, %s)", c.quoteValue(key), c.quoteValue(val))
	}

	// Simple @ events
	eventMap := map[string]string{
		"click":      "AtClick",
		"submit":     "AtSubmit",
		"change":     "AtChange",
		"input":      "AtInput",
		"keydown":    "AtKeydown",
		"keyup":      "AtKeyup",
		"mouseenter": "AtMouseenter",
		"mouseleave": "AtMouseleave",
	}

	if funcName, ok := eventMap[event]; ok {
		return fmt.Sprintf("alpine.%s(%s)", funcName, c.quoteValue(val))
	}

	// Generic @ event
	return fmt.Sprintf("alpine.At(%s, %s)", c.quoteValue(event), c.quoteValue(val))
}

// convertAlpineBindAttribute converts Alpine : bind attributes
func (c *Converter) convertAlpineBindAttribute(key, val string) string {
	// Remove : prefix
	attr := strings.TrimPrefix(key, ":")

	// Common bind attributes
	bindMap := map[string]string{
		"class":    "ColonClass",
		"style":    "ColonStyle",
		"disabled": "ColonDisabled",
		"value":    "ColonValue",
		"key":      "Colon",
	}

	if funcName, ok := bindMap[attr]; ok {
		if funcName == "Colon" {
			return fmt.Sprintf("alpine.Colon(%s, %s)", c.quoteValue(attr), c.quoteValue(val))
		}
		return fmt.Sprintf("alpine.%s(%s)", funcName, c.quoteValue(val))
	}

	// Generic : bind
	return fmt.Sprintf("alpine.Colon(%s, %s)", c.quoteValue(attr), c.quoteValue(val))
}

// containsMultilineContent checks if args should be formatted on multiple lines
func containsMultilineContent(args []string) bool {
	if len(args) > 5 {
		return true
	}

	totalLen := 0
	for _, arg := range args {
		totalLen += len(arg)
		if strings.Contains(arg, "\n") {
			return true
		}
	}

	return totalLen > 80
}
