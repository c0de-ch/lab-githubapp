package main

import "embed"

//go:embed web/templates
var embeddedTemplateFS embed.FS

//go:embed web/static
var embeddedStaticFS embed.FS
