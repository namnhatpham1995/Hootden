# pages Specification

## Purpose
Pages are what a person actually writes in: a nested tree of documents inside a workspace, each holding rich text that saves as they type. The tree gives structure for organising notes and plans; the document gives checklists, headings, and prose without needing any separate task or outline feature.

## Requirements

### Requirement: Create a page

The system SHALL allow a signed-in person to create a page in their workspace, either at the top level of the tree or as a child of an existing page. A newly created page SHALL have an empty document and SHALL be placed last among its siblings.

A page SHALL have a title. A page created without one SHALL be given a placeholder title rather than an empty one, so it is identifiable in the tree.

#### Scenario: Create a top-level page

- **WHEN** a person creates a page with no parent specified
- **THEN** a page with an empty document appears at the end of the top level of their tree and opens for editing

#### Scenario: Create a nested page

- **WHEN** a person creates a page as a child of an existing page
- **THEN** the new page appears as the last child of that parent, and the parent is shown as having children

#### Scenario: Untitled page is still identifiable

- **WHEN** a person creates a page and does not enter a title
- **THEN** the page is listed under a placeholder title rather than a blank entry

### Requirement: Browse the page tree

The system SHALL present the pages of a workspace as a tree that reflects parent-child nesting and sibling order. Nesting depth SHALL NOT be limited by the system.

#### Scenario: Tree reflects structure and order

- **WHEN** a person opens their workspace
- **THEN** their pages are listed in tree form, with each page's children beneath it in their stored order

#### Scenario: Empty workspace

- **WHEN** a person whose workspace contains no pages opens it
- **THEN** they are shown an empty state that offers to create their first page, rather than a blank tree

### Requirement: Rename a page

The system SHALL allow a person to change a page's title, and SHALL reflect the new title in the tree and anywhere else the page is named.

#### Scenario: Rename updates every reference

- **WHEN** a person changes a page's title
- **THEN** the tree entry and the page's own heading both show the new title without a reload

### Requirement: Reposition a page

The system SHALL allow a person to move a page to a different position among its siblings, and to move it under a different parent or to the top level. Moving a page SHALL move its descendants with it, preserving their structure.

The system SHALL reject a move that would make a page a descendant of itself.

#### Scenario: Reorder among siblings

- **WHEN** a person moves a page to a different position among its current siblings
- **THEN** the tree shows it in the new position, and that order persists across reloads

#### Scenario: Reparent a page with its children

- **WHEN** a person moves a page that has children under a different parent
- **THEN** the page and its whole subtree appear under the new parent with their nesting intact

#### Scenario: Cyclic move is refused

- **WHEN** a person attempts to move a page under one of its own descendants
- **THEN** the system refuses the move and the tree is left unchanged

### Requirement: Delete a page

The system SHALL allow a person to delete a page. Deleting a page SHALL permanently delete its descendants as well. Because deletion is permanent and there is no trash in this version, the system MUST require an explicit confirmation that states how many pages will be removed.

#### Scenario: Deleting a leaf page

- **WHEN** a person confirms deletion of a page with no children
- **THEN** the page and its document are permanently removed and it disappears from the tree

#### Scenario: Deleting a page with descendants warns first

- **WHEN** a person deletes a page that has descendants
- **THEN** the confirmation states how many pages will be permanently removed, and only on confirmation are the page and all its descendants deleted

#### Scenario: Cancelling deletion changes nothing

- **WHEN** a person dismisses the deletion confirmation
- **THEN** no page is removed

### Requirement: Edit page content as a rich document

The system SHALL let a person edit a page's body as a rich text document and SHALL preserve its structure, not only its plain text.

The document SHALL support at minimum: paragraphs, headings of at least three levels, bulleted lists, numbered lists, checkbox lists whose items can be checked and unchecked, code blocks, block quotes, horizontal rules, and inline bold, italic, strikethrough, inline code, and links.

Checkbox state SHALL persist with the document, so a page can serve as a to-do list.

#### Scenario: Structure survives a reload

- **WHEN** a person writes a document using headings, nested lists, and inline formatting, and later reopens the page
- **THEN** the document is restored with the same structure and formatting

#### Scenario: Checking an item persists

- **WHEN** a person checks an item in a checkbox list and reopens the page later
- **THEN** the item is still checked

### Requirement: Edits save automatically

The system SHALL save document changes without the person invoking a save action. Changes SHALL be persisted within 2 seconds of the person pausing.

The system SHALL show whether current changes are saved, saving, or failed to save. On a failed save the system MUST keep the unsaved changes in the editor and retry rather than discarding them silently.

Moving from one page to another inside the application counts as navigating away for the purposes of this requirement. Unsaved changes SHALL be persisted before the editor is left, whether the person closes the tab, follows a link out, or selects a different page in the tree. Where they cannot be persisted, the person SHALL be warned before the changes are discarded. A scheduled or retrying save MUST NOT be abandoned without either completing or warning.

A save that fails in a way the same request will always fail SHALL stop retrying and report itself as needing attention. The system SHALL distinguish these from failures that a later attempt could succeed at — an unreachable API, a timeout, a server error — which SHALL continue to be retried. In neither case are the changes discarded from the editor.

#### Scenario: Pausing saves the work

- **WHEN** a person types in a document and then pauses
- **THEN** the changes are persisted within 2 seconds and the interface indicates the page is saved

#### Scenario: Leaving after the saved indicator keeps the work

- **WHEN** a person waits for the saved indication and then navigates away or closes the tab
- **THEN** reopening the page shows the work they left

#### Scenario: Save failure does not lose work

- **WHEN** a save fails because the API is unreachable
- **THEN** the interface shows that saving failed, the person's changes remain in the editor, and saving is retried when the API is reachable again

#### Scenario: Navigating away with unsaved changes warns

- **WHEN** a person tries to close or navigate away while changes are still unsaved
- **THEN** they are warned before leaving

#### Scenario: Selecting another page persists the pending edit

- **WHEN** a person types in a document and selects a different page in the tree before the pending save has run
- **THEN** the pending changes are persisted, and reopening the first page shows them

#### Scenario: Leaving during a retry does not silently drop the work

- **WHEN** a person's save has failed and is awaiting a retry, and they select a different page
- **THEN** the system either completes the save or warns them that the changes will be lost, and never discards them without either

#### Scenario: A save that can never succeed stops retrying

- **WHEN** a save is rejected for a reason that will recur on every identical attempt, such as a document the API will not accept
- **THEN** the system stops retrying, tells the person that the document could not be saved and why, and leaves their changes in the editor

#### Scenario: Session lost while editing keeps the text on screen

- **WHEN** a person's session ends while they have unsaved changes and the next save is refused as unauthenticated
- **THEN** their text remains on screen and readable, they are told they have been signed out, and the system does not navigate away from the unsaved content on its own

### Requirement: Concurrent edits from one account resolve predictably

Because this version has no collaborative editing, the system SHALL resolve simultaneous edits to the same page by accepting the most recent write. The system MUST NOT merge concurrent documents or silently interleave them.

#### Scenario: Two tabs editing one page

- **WHEN** the same person edits one page in two tabs at once
- **THEN** the document that saved most recently is the one stored, and reopening the page shows that document rather than a merged result
