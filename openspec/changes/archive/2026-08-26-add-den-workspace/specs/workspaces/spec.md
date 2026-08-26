## Purpose

A workspace is the container that owns pages and defines who may reach them. In this version the only kind that exists is the Den: a personal workspace belonging to exactly one account, created automatically so that a newly signed-in person has somewhere to write immediately.

## ADDED Requirements

### Requirement: Every account has exactly one Den

The system SHALL create a personal workspace — a Den — when an account is created, and SHALL ensure each account has exactly one. A person SHALL NOT be able to create additional workspaces, and SHALL NOT be able to delete their Den.

A workspace SHALL record whether it is personal. In this version every workspace created is personal.

#### Scenario: Den exists on first sign-in

- **WHEN** a person signs in for the first time and their account is created
- **THEN** a personal workspace is created for that account and they land in it with no setup step

#### Scenario: Returning sign-in reuses the same Den

- **WHEN** an existing person signs in again
- **THEN** they land in the same Den holding the pages they left, and no second workspace is created

#### Scenario: No workspace management surface

- **WHEN** a signed-in person uses the app
- **THEN** no option is offered to create, rename, switch, leave, or delete a workspace

### Requirement: Den contents are reachable only by its owner

The system SHALL permit access to a workspace and everything it contains only to the account that owns it. Ownership SHALL be checked on the server for every request that reads or changes workspace contents, and MUST NOT rely on the client having withheld an identifier.

#### Scenario: Owner reaches their own Den

- **WHEN** a signed-in person requests their own Den or its contents
- **THEN** the system serves the request

#### Scenario: Another account's Den is not reachable

- **WHEN** a signed-in person requests a workspace, or a page inside a workspace, owned by a different account
- **THEN** the system refuses the request and discloses nothing about whether that workspace or page exists

#### Scenario: Ownership is enforced on writes, not only reads

- **WHEN** a signed-in person attempts to create, change, move, or delete a page in a workspace they do not own
- **THEN** the system refuses the request and the target workspace is left unchanged
