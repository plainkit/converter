package main

import (
	"strings"
	"testing"
)

func TestConvertFullPage(t *testing.T) {
	input := `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>Test Page</title>
</head>
<body>
	<div class="container">
		<h1>Welcome</h1>
		<p>This is a test page.</p>
	</div>
</body>
</html>`

	expected := []string{
		"func Page() Node",
		"Html(",
		"Head(",
		`Meta(Charset("utf-8"))`,
		`HeadTitle(T("Test Page"))`,
		"Body(",
		`Class("container")`,
		`T("Welcome")`,
		`T("This is a test page.")`,
	}

	converter := NewConverter(false, false)
	result, err := converter.Convert(input)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", exp, result)
		}
	}

	// Check imports
	if !strings.Contains(result, `"github.com/plainkit/html"`) {
		t.Error("Expected html import to be present")
	}

	// Check function signature
	if !strings.Contains(result, "func Page() Node") {
		t.Error("Expected Page function for full documents")
	}
}

func TestConvertMultipleFragments(t *testing.T) {
	input := `<div>Fragment 1</div><p>Fragment 2</p>`

	expected := []string{
		"func Components() []Node",
		"return []Node{",
		"Div(",
		`T("Fragment 1")`,
		"P(",
		`T("Fragment 2")`,
	}

	converter := NewConverter(false, false)
	result, err := converter.Convert(input)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", exp, result)
		}
	}

	// Multiple fragments should return Components() []Node
	if !strings.Contains(result, "func Components() []Node") {
		t.Error("Expected Components function for multiple fragments")
	}
}

func TestConvertBasicHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Check if output contains these strings
	}{
		{
			name:  "Simple div with class",
			input: `<div class="container">Hello World</div>`,
			expected: []string{
				"Div(",
				`Class("container")`,
				`T("Hello World")`,
			},
		},
		{
			name:  "Nested elements",
			input: `<div><p>Paragraph <strong>bold</strong> text</p></div>`,
			expected: []string{
				"Div(",
				"P(",
				`T("Paragraph")`,
				"Strong(",
				`T("bold")`,
				`T("text")`,
			},
		},
		{
			name: "Form with inputs",
			input: `<form method="post" action="/submit">
				<input type="text" name="username" placeholder="Username">
				<button type="submit">Submit</button>
			</form>`,
			expected: []string{
				"Form(",
				`Method("post")`,
				`Action("/submit")`,
				"Input(",
				`InputType("text")`,
				`InputName("username")`,
				`Placeholder("Username")`,
				"Button(",
				`ButtonType("submit")`,
				`T("Submit")`,
			},
		},
		{
			name:  "Link with href",
			input: `<a href="https://example.com">Click here</a>`,
			expected: []string{
				"A(",
				`Href("https://example.com")`,
				`T("Click here")`,
			},
		},
		{
			name:  "Image with attributes",
			input: `<img src="image.jpg" alt="Description" width="300" height="200">`,
			expected: []string{
				"Img(",
				`Src("image.jpg")`,
				`Alt("Description")`,
				`Width("300")`,
				`Height("200")`,
			},
		},
		{
			name: "List structure",
			input: `<ul>
				<li>Item 1</li>
				<li>Item 2</li>
			</ul>`,
			expected: []string{
				"Ul(",
				"Li(",
				`T("Item 1")`,
				`T("Item 2")`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewConverter(false, false)
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Conversion failed: %v", err)
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", expected, result)
				}
			}

			// Check imports
			if !strings.Contains(result, `"github.com/plainkit/html"`) {
				t.Error("Expected html import to be present")
			}

			// Check function signature
			if !strings.Contains(result, "func Component() Node") {
				t.Error("Expected Component function for fragments")
			}
		})
	}
}

func TestConvertHTMXAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "HTMX GET request",
			input: `<button hx-get="/api/data" hx-target="#result">Load</button>`,
			expected: []string{
				"Button(",
				`htmx.HxGet("/api/data")`,
				`htmx.HxTarget("#result")`,
				`T("Load")`,
			},
		},
		{
			name:  "HTMX POST with swap",
			input: `<form hx-post="/api/submit" hx-swap="innerHTML">`,
			expected: []string{
				"Form(",
				`htmx.HxPost("/api/submit")`,
				`htmx.HxSwap("innerHTML")`,
			},
		},
		{
			name:  "HTMX trigger and indicator",
			input: `<div hx-get="/data" hx-trigger="load" hx-indicator=".loading">Content</div>`,
			expected: []string{
				"Div(",
				`htmx.HxGet("/data")`,
				`htmx.HxTrigger("load")`,
				`htmx.HxIndicator(".loading")`,
				`T("Content")`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewConverter(true, false) // Enable HTMX
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Conversion failed: %v", err)
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", expected, result)
				}
			}

			// Check htmx import
			if !strings.Contains(result, `"github.com/plainkit/htmx"`) {
				t.Error("Expected htmx import to be present")
			}
		})
	}
}

func TestConvertAlpineAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "Alpine x-data and x-show",
			input: `<div x-data="{ open: false }" x-show="open">Content</div>`,
			expected: []string{
				"Div(",
				`alpine.XData("{ open: false }")`,
				`alpine.XShow("open")`,
				`T("Content")`,
			},
		},
		{
			name:  "Alpine @ event handlers",
			input: `<button @click="count++" @keydown.enter="submit()">Click</button>`,
			expected: []string{
				"Button(",
				`alpine.AtClick("count++")`,
				`alpine.AtKeydownEnter("submit()")`,
				`T("Click")`,
			},
		},
		{
			name:  "Alpine : bindings",
			input: `<div :class="{ 'active': isActive }" :style="styles">Text</div>`,
			expected: []string{
				"Div(",
				`alpine.ColonClass("{ 'active': isActive }")`,
				`alpine.ColonStyle("styles")`,
				`T("Text")`,
			},
		},
		{
			name:  "Alpine x-model",
			input: `<input type="text" x-model="username">`,
			expected: []string{
				"Input(",
				`InputType("text")`,
				`alpine.XModel("username")`,
			},
		},
		{
			name:  "Alpine x-for template",
			input: `<template x-for="item in items"><li x-text="item"></li></template>`,
			expected: []string{
				"Template(",
				`alpine.XFor("item in items")`,
				"Li(",
				`alpine.XText("item")`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewConverter(false, true) // Enable Alpine
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Conversion failed: %v", err)
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", expected, result)
				}
			}

			// Check alpine import
			if !strings.Contains(result, `"github.com/plainkit/alpine"`) {
				t.Error("Expected alpine import to be present")
			}
		})
	}
}

func TestConvertCombinedHTMXAndAlpine(t *testing.T) {
	input := `<div x-data="{ loading: false }" hx-get="/api/data" @htmx:before-request="loading = true" @htmx:after-request="loading = false">
		<button @click="$htmx.trigger('load')" hx-target="#result">Load Data</button>
		<div x-show="loading">Loading...</div>
		<div id="result"></div>
	</div>`

	expected := []string{
		`alpine.XData("{ loading: false }")`,
		`htmx.HxGet("/api/data")`,
		`alpine.At("htmx:before-request", "loading = true")`,
		`alpine.At("htmx:after-request", "loading = false")`,
		`alpine.AtClick("$htmx.trigger('load')")`,
		`htmx.HxTarget("#result")`,
		`alpine.XShow("loading")`,
		`Id("result")`,
		`"github.com/plainkit/htmx"`,
		`"github.com/plainkit/alpine"`,
	}

	converter := NewConverter(true, true) // Enable both
	result, err := converter.Convert(input)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", exp, result)
		}
	}
}

func TestConvertSpecialAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "Data attributes",
			input: `<div data-id="123" data-name="test">Content</div>`,
			expected: []string{
				`Data("id", "123")`,
				`Data("name", "test")`,
			},
		},
		{
			name:  "ARIA attributes",
			input: `<button aria-label="Close" aria-expanded="false" role="button">X</button>`,
			expected: []string{
				`Aria("label", "Close")`,
				`Aria("expanded", "false")`,
				`Role("button")`,
			},
		},
		{
			name:  "Boolean attributes",
			input: `<input type="checkbox" checked disabled required>`,
			expected: []string{
				"Checked()",
				"Disabled()",
				"Required()",
			},
		},
		{
			name:  "Meta tags",
			input: `<meta charset="utf-8"><meta name="viewport" content="width=device-width">`,
			expected: []string{
				`Charset("utf-8")`,
				`Name("viewport")`,
				`Content("width=device-width")`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewConverter(false, false)
			result, err := converter.Convert(tt.input)
			if err != nil {
				t.Fatalf("Conversion failed: %v", err)
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, but it doesn't.\nOutput:\n%s", expected, result)
				}
			}
		})
	}
}
