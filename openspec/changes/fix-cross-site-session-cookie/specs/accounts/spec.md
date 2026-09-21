## ADDED Requirements

### Requirement: A deployment that cannot deliver the session fails loudly

The system SHALL refuse to serve requests when its own configuration makes it impossible for the browser to return the session cookie to the API, and SHALL report that refusal at startup rather than at sign-in.

A configuration is impossible in this sense whenever the cookie the API issues would not be sent by a browser on a credentialed request originating from the configured web app origin — including when the web app and the API are served from hosts that do not share a registrable domain, and when the cookie's configured domain scope does not cover both hosts.

The refusal MUST name the conflicting configuration values, so that an operator can correct them without reproducing the failure in a browser.

The system MUST NOT satisfy this requirement by widening the cookie's scope: relaxing the session cookie to a form browsers treat as third-party is a violation of "Session cookie is usable across the app and API origins", not a resolution of this one.

#### Scenario: Web app and API on unrelated domains is refused at startup

- **WHEN** the system starts configured with a web app origin and an API host that share no registrable domain, so a session cookie issued by the API would never be sent from the web app
- **THEN** the system refuses to start and reports which configured values conflict, and no sign-in attempt is ever served

#### Scenario: Cookie scope that excludes the web app is refused at startup

- **WHEN** the system starts configured with a cookie domain scope that does not cover the configured web app origin's host
- **THEN** the system refuses to start and reports which configured values conflict

#### Scenario: Shared-parent-domain deployment starts normally

- **WHEN** the system starts configured with the web app at a domain apex and the API at a subdomain of that same domain, and a cookie domain scope covering both
- **THEN** the system starts and serves requests

#### Scenario: Single-host deployment starts normally

- **WHEN** the system starts configured with the web app and the API on the same host and no cookie domain scope set, so the session cookie is host-only and reaches both
- **THEN** the system starts and serves requests

#### Scenario: Misconfiguration is not reported as a failed sign-in

- **WHEN** a deployment is configured such that the session cookie cannot reach the API from the web app
- **THEN** the failure surfaces as a refusal to start, and never as a registration or sign-in that reports success while leaving the person signed out
