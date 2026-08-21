// Package page manages the page tree: creation, the tree listing,
// renaming, moving, and deletion. Document content (task group 6) is
// handled elsewhere.
package page

// Node is a page as it appears in the tree listing -- deliberately without
// the doc body, which the tree endpoint never returns.
type Node struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id,omitempty"`
	Title    string `json:"title"`
	Position int    `json:"position"`
}
