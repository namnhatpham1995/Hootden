## MODIFIED Requirements

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

### Requirement: Protected resources require a session

The system SHALL reject unauthenticated requests to any resource other than the landing page, the sign-in entry point, the provider callback, registration, and login.

#### Scenario: Unauthenticated API request is refused

- **WHEN** a request without a valid session asks the API for workspace or page data
- **THEN** the system refuses it with an unauthenticated response and returns no workspace or page content

#### Scenario: Unauthenticated visitor to the app is sent to sign in

- **WHEN** a person without a valid session opens a page inside the app
- **THEN** they are shown the signed-out landing state with a way to sign in or register

## ADDED Requirements

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
