## ADDED Requirements

### Requirement: Sign-in state that cannot be determined is reported

The system SHALL treat "not signed in" and "unable to determine whether you are signed in" as distinct states, and SHALL present the second one to the person rather than resolving it silently to either the signed-out state or an empty page.

When the check fails for a reason unrelated to authentication — the API being unreachable, a timeout, a server error — the system SHALL say that it could not load the account and SHALL offer a way to try again. It MUST NOT leave the interface blank, and MUST NOT present the person as signed out, since doing so invites them to sign in again against an API that cannot answer.

#### Scenario: API error while checking sign-in state is reported

- **WHEN** the check for the current account fails with a server or network error
- **THEN** the person is shown that the account could not be loaded, together with a way to retry

#### Scenario: A failed check is not shown as being signed out

- **WHEN** the check for the current account fails for a reason other than the absence of a valid session
- **THEN** the system does not present the signed-out landing state

#### Scenario: Genuinely signed out is still shown as signed out

- **WHEN** the check for the current account completes and reports no valid session
- **THEN** the person is shown the signed-out landing state with a way to sign in or register
