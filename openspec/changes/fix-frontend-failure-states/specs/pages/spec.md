## MODIFIED Requirements

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
