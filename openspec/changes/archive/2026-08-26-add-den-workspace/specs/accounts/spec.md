## Purpose

Establishes who a person is and keeps them signed in across visits, using a third-party identity provider so that Hootden never handles passwords. Sessions must survive the split between the web app and the API, which are served from different subdomains of one registrable domain.

## ADDED Requirements

### Requirement: Sign in with Google

The system SHALL allow a person to sign in using a Google account, and SHALL treat sign-in and sign-up as the same action. Google is the only supported identity provider.

The system SHALL create an account on first successful sign-in, recording the provider's stable subject identifier and the email address it reports. Subsequent sign-ins with the same subject identifier SHALL resolve to the existing account rather than creating a new one.

#### Scenario: First sign-in creates an account

- **WHEN** a person completes Google sign-in with an account Hootden has not seen before
- **THEN** the system creates a new account, establishes a session, and lands them in their own workspace

#### Scenario: Returning sign-in reuses the account

- **WHEN** a person completes Google sign-in with an account that already exists
- **THEN** the system establishes a session for the existing account and creates no new account or workspace

#### Scenario: Person declines consent at the provider

- **WHEN** a person cancels or denies the Google consent screen
- **THEN** the system returns them to the signed-out landing state with an explanation, and creates no account

#### Scenario: Provider callback cannot be verified

- **WHEN** the sign-in callback arrives with a missing, expired, or mismatched state value, or with credentials the provider will not validate
- **THEN** the system rejects the sign-in, establishes no session, and does not reveal whether any matching account exists

### Requirement: Session cookie is usable across the app and API origins

The system SHALL deliver the session to the browser in a cookie whose attributes allow it to be sent to both the web app at the domain apex and the API at its `api.` subdomain, without relying on cross-site cookie delivery.

The cookie MUST be scoped to the shared parent domain, MUST be marked `HttpOnly` and `Secure`, and MUST use a `SameSite` value that browsers deliver on same-site subdomain requests without third-party cookie permission. The cookie MUST NOT require `SameSite=None`.

The cookie value MUST be an opaque token that carries no account information a client could read or alter.

#### Scenario: Authenticated request from the web app reaches the API

- **WHEN** the signed-in web app served from the domain apex makes a credentialed request to the API on the `api.` subdomain
- **THEN** the browser sends the session cookie and the API resolves the request to the signed-in account

#### Scenario: Browser blocking third-party cookies still keeps the session

- **WHEN** a signed-in person uses a browser configured to block third-party cookies
- **THEN** their session continues to work for both the web app and the API

#### Scenario: Session token is not readable by page scripts

- **WHEN** scripts running on the page attempt to read the session cookie
- **THEN** the cookie is not exposed to them

### Requirement: Sessions expire and can be verified

The system SHALL treat a session as valid only while it has not expired and has not been revoked. Every request to a protected resource SHALL be resolved against server-held session state rather than trusting the cookie's contents alone.

An idle session SHALL expire no later than 30 days after it was issued.

#### Scenario: Valid session resolves to its account

- **WHEN** a request arrives carrying an unexpired, unrevoked session cookie
- **THEN** the system resolves it to the owning account and serves the request

#### Scenario: Expired session is rejected

- **WHEN** a request arrives carrying a session cookie whose session has passed its expiry
- **THEN** the system rejects the request as unauthenticated and clears the stale cookie

#### Scenario: Forged or unknown token is rejected

- **WHEN** a request arrives carrying a session cookie value that matches no stored session
- **THEN** the system rejects the request as unauthenticated

### Requirement: Sign out

The system SHALL let a signed-in person end their session. Signing out MUST revoke the session on the server, not only remove the cookie from the browser.

#### Scenario: Signing out ends the session everywhere

- **WHEN** a signed-in person signs out
- **THEN** the session is revoked, the cookie is cleared, and any later request presenting that same token is rejected as unauthenticated

### Requirement: Protected resources require a session

The system SHALL reject unauthenticated requests to any resource other than the landing page, the sign-in entry point, and the provider callback.

#### Scenario: Unauthenticated API request is refused

- **WHEN** a request without a valid session asks the API for workspace or page data
- **THEN** the system refuses it with an unauthenticated response and returns no workspace or page content

#### Scenario: Unauthenticated visitor to the app is sent to sign in

- **WHEN** a person without a valid session opens a page inside the app
- **THEN** they are shown the signed-out landing state with a way to sign in
