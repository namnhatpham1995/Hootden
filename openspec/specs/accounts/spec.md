# accounts Specification

## Purpose
Establishes who a person is and keeps them signed in across visits, using a third-party identity provider so that Hootden never handles passwords. Sessions must survive the split between the web app and the API, which are served from different subdomains of one registrable domain.

## Requirements

### Requirement: Sign in with Google

The system SHALL allow a person to sign in using a Google account, and SHALL treat sign-in and sign-up as the same action. Google is one of two supported identity methods, alongside email and password.

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

#### Scenario: Google sign-in is not configured

- **WHEN** the deployment has no Google OAuth client configured
- **THEN** the system does not offer Google sign-in, and email/password sign-in remains available

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

The system SHALL reject unauthenticated requests to any resource other than the landing page, the sign-in entry point, the provider callback, registration, and login.

#### Scenario: Unauthenticated API request is refused

- **WHEN** a request without a valid session asks the API for workspace or page data
- **THEN** the system refuses it with an unauthenticated response and returns no workspace or page content

#### Scenario: Unauthenticated visitor to the app is sent to sign in

- **WHEN** a person without a valid session opens a page inside the app
- **THEN** they are shown the signed-out landing state with a way to sign in or register

### Requirement: Register with email and password

The system SHALL allow a person to create an account with an email address and a password, without any third-party identity provider. Registration and the first session it creates SHALL happen as a single action.

The email address MUST be unique across every account, regardless of which method created it. The password MUST be at least 8 characters. The system SHALL store only a salted hash of the password, never the password itself.

#### Scenario: Registration with a new email creates an account

- **WHEN** a person registers with an email address no account uses and a password meeting the minimum length
- **THEN** the system creates a new account, establishes a session, and lands them in their own workspace

#### Scenario: Registration with an email already in use is rejected

- **WHEN** a person registers with an email address that already belongs to an account, whether created by Google sign-in or by password registration
- **THEN** the system rejects the registration, creates no account, and does not establish a session

#### Scenario: Registration with a too-short password is rejected

- **WHEN** a person submits a password shorter than the minimum length
- **THEN** the system rejects the registration before storing anything

### Requirement: Sign in with email and password

The system SHALL allow a person with a password-registered account to establish a session by presenting their email address and password.

#### Scenario: Correct credentials establish a session

- **WHEN** a person submits the email and password of an existing password-registered account
- **THEN** the system establishes a session for that account

#### Scenario: Incorrect password is rejected

- **WHEN** a person submits an email address that has an account and a password that does not match it
- **THEN** the system rejects the sign-in, establishes no session, and does not reveal whether the password was close to correct

#### Scenario: Unknown email is rejected the same way as a wrong password

- **WHEN** a person submits an email address with no matching account
- **THEN** the system rejects the sign-in identically to a wrong-password rejection, without revealing that no account exists

#### Scenario: Signing in to a Google-only account with a password is rejected

- **WHEN** a person submits an email and password for an account that was created via Google sign-in and has no password set
- **THEN** the system rejects the sign-in identically to a wrong-password rejection

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
