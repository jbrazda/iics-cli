// Package dependencies works out which IICS objects belong together. It
// resolves seed assets from user input or a build manifest, walks object
// references to build a dependency graph, and expands a selection to the
// closure of everything it references.
package dependencies
