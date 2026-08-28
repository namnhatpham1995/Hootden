## ADDED Requirements

### Requirement: Repeated authentication attempts are limited

The system SHALL limit how many authentication attempts it will process for a given account and for a given source over a period of time, and SHALL refuse further attempts once that limit is reached.

The limit MUST apply to sign-in and to registration. It MUST bound both the guessing of a single account's password and the volume of unauthenticated work any one caller can compel the system to perform, since every attempt is deliberately expensive to evaluate.

A refusal for exceeding the limit SHALL be distinguishable by the person from a rejected credential, so that someone who has mistyped their own password is not left believing it is wrong. It MUST NOT reveal whether the email address belongs to an account: attempts against an account that exists and one that does not SHALL be limited identically.

A refusal SHALL be temporary and SHALL lift on its own without an administrator, and correct credentials presented after it lifts SHALL establish a session normally.

#### Scenario: Repeated wrong passwords are eventually refused

- **WHEN** sign-in is attempted for one account with an incorrect password more times than the limit allows
- **THEN** further attempts are refused without evaluating the credential, and the person is told they have attempted too many times rather than that the password was wrong

#### Scenario: The limit does not reveal whether an account exists

- **WHEN** the limit is reached by attempts against an email address with no account
- **THEN** the refusal is indistinguishable from the refusal for an address that does have one

#### Scenario: A refusal lifts on its own

- **WHEN** a person waits out a refusal and then presents correct credentials
- **THEN** the system establishes a session, with no administrative action required

#### Scenario: One caller cannot compel unbounded work

- **WHEN** a single source submits authentication attempts faster than the limit allows, whatever addresses they name
- **THEN** the system refuses the excess without evaluating the submitted credentials

#### Scenario: An ordinary mistyped password is unaffected

- **WHEN** a person mistypes their password a small number of times and then enters it correctly
- **THEN** they sign in normally, having never been refused

### Requirement: Expired sessions are removed

The system SHALL remove session records once they have expired, rather than retaining them indefinitely. Removal is housekeeping, not a change to when a session stops being usable: an expired session is already refused, and removal SHALL NOT be what makes it so.

Removal SHALL happen without an operator invoking it, and SHALL NOT interrupt serving requests.

#### Scenario: Expired sessions do not accumulate

- **WHEN** sessions pass their expiry and are never explicitly signed out
- **THEN** their records are removed without anyone acting

#### Scenario: Removal does not affect a live session

- **WHEN** expired sessions are removed while other sessions are still within their lifetime
- **THEN** those sessions continue to resolve to their accounts

#### Scenario: An expired session is refused whether or not it has been removed

- **WHEN** a request presents a session cookie whose session has expired but whose record has not yet been removed
- **THEN** the request is refused as unauthenticated

### Requirement: State changes are not reachable by GET

The system SHALL NOT expose any operation that creates, modifies, or deletes data as an HTTP `GET`. This is load-bearing rather than stylistic: the session cookie is delivered with a `SameSite` value that browsers send on top-level cross-site `GET` navigation, so a mutation reachable by `GET` would be reachable cross-site.

The provider sign-in callback is the sole exception, and is exempt only because the protocol requires it: an identity provider returns the person by redirecting their browser, which is a `GET`, and it creates an account and a session. It SHALL therefore carry its own protection — the callback MUST verify a value it issued at the start of the flow, held in a cookie confined to the callback's own host, and MUST reject any callback that cannot be matched to a sign-in this system began, before performing any exchange or creating anything.

Any future exception SHALL be stated in this requirement together with the protection that makes it safe. An endpoint that changes state on `GET` and is not named here is a defect.

#### Scenario: Every mutation requires a non-GET method

- **WHEN** the system's routes are audited
- **THEN** every operation that creates, modifies, or deletes data is reachable only by `POST`, `PATCH`, or `DELETE`, with no exception other than the provider callback

#### Scenario: An unsolicited provider callback creates nothing

- **WHEN** a callback arrives that cannot be matched to a sign-in this system started, including one a third-party site caused the browser to make
- **THEN** it is rejected before any credential exchange, and no account and no session are created

#### Scenario: A cross-site GET cannot change anything

- **WHEN** a third-party site causes a person's browser to make a top-level `GET` request to any of the system's endpoints
- **THEN** no data is created, modified, or deleted as a result
