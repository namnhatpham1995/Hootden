## ADDED Requirements

### Requirement: Password entry can be revealed

The system SHALL let a person reveal the password they are entering, and re-mask it, at any point while entering it. Every field that accepts a password SHALL offer this, on both registration and sign-in.

Password fields SHALL remain masked until the person asks for them to be revealed. Revealing MUST NOT submit the form, alter what was typed, or persist beyond the current entry.

#### Scenario: Revealing shows what was typed

- **WHEN** a person entering a password asks for it to be revealed
- **THEN** the characters they have typed become readable, and what will be submitted is unchanged

#### Scenario: Revealing does not submit the form

- **WHEN** a person asks to reveal a password before completing the form
- **THEN** no registration or sign-in is attempted

#### Scenario: Fields start masked

- **WHEN** a person opens the registration or sign-in form
- **THEN** every password field is masked until they ask otherwise

#### Scenario: Revealed state does not outlive the entry

- **WHEN** a person reveals a password and then leaves or reloads the form
- **THEN** the next entry begins masked

## MODIFIED Requirements

### Requirement: Register with email and password

The system SHALL allow a person to create an account with an email address and a password, without any third-party identity provider. Registration and the first session it creates SHALL happen as a single action.

The email address MUST be unique across every account, regardless of which method created it, and MUST be a well-formed email address. The password MUST be at least 8 characters, and MUST NOT exceed the longest password the system can hash. The system SHALL store only a salted hash of the password, never the password itself.

Every limit on a submitted credential SHALL be enforced as a stated rejection that names what was wrong, never as an internal error. A password the system cannot hash is a rejected input, not a failure of the system.

Registration SHALL require the password to be entered twice and SHALL reject the registration when the two entries differ. The confirming entry exists only to catch a typing mistake: it MUST NOT be stored, and MUST NOT be sent to the API, which continues to receive a single password. Rejecting a mismatch SHALL leave what was already typed in place rather than clearing the form.

Sign-in SHALL NOT ask for a confirming entry — there is an existing password to check against, so a second entry protects nothing there.

#### Scenario: Registration with a new email creates an account

- **WHEN** a person registers with an email address no account uses and a password meeting the minimum length, entered identically twice
- **THEN** the system creates a new account, establishes a session, and lands them in their own workspace

#### Scenario: Registration with an email already in use is rejected

- **WHEN** a person registers with an email address that already belongs to an account, whether created by Google sign-in or by password registration
- **THEN** the system rejects the registration, creates no account, and does not establish a session

#### Scenario: Registration with a too-short password is rejected

- **WHEN** a person submits a password shorter than the minimum length
- **THEN** the system rejects the registration before storing anything

#### Scenario: Registration with an over-long password is rejected, not failed

- **WHEN** a person submits a password longer than the system is able to hash
- **THEN** the system rejects the registration as invalid input, tells them the password is too long, creates no account, and does not report an internal error

#### Scenario: Registration with mismatched password entries is rejected

- **WHEN** a person submits a registration whose two password entries differ
- **THEN** the system rejects it, explains that the entries do not match, creates no account, and leaves the email and both entries as they were typed

#### Scenario: Registration with a malformed email is rejected by the API

- **WHEN** a registration request reaches the API carrying an email address that is empty or not a well-formed address, whatever the browser allowed
- **THEN** the system rejects it, creates no account, and establishes no session
