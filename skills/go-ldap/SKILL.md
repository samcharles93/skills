---
name: go-ldap
description: This skill should be used when the user works with LDAP or Active Directory from Go, or says "bind to LDAP", "search LDAP", "authenticate against Active Directory", "reset a user password in AD", "add a user to a group", "LDAP over TLS", "StartTLS", "search the directory", or mentions go-ldap, ldap/v3, baseDN, bindDN, sAMAccountName, userPrincipalName, distinguishedName, unicodePwd or an LDAP filter. Covers client setup, user and group operations, authentication, password changes, computer objects, LDAP error handling, and integration testing with Testcontainers.
---

# Go LDAP and Active Directory

Working with directory servers from Go using `github.com/go-ldap/ldap/v3`.

## When to Use

Use this skill when Go code needs to talk to LDAP or Active Directory:

- Connecting and binding, with TLS or StartTLS
- Searching for users, groups or computer objects
- Authenticating a user against the directory
- Changing or resetting passwords, including the Active Directory `unicodePwd` rules
- Managing group membership
- Interpreting LDAP result codes rather than matching on error text
- Testing against a real directory with Testcontainers

## Reference

`references/ldap.md` carries the working patterns, organised as:

| Section | Covers |
|---|---|
| Client Setup | Connection, bind, TLS and StartTLS, client struct |
| User Operations | Search, filters, attribute retrieval, creating and updating users |
| Authentication | Bind-based authentication and its failure modes |
| Password Operations | Setting and resetting passwords, Active Directory encoding rules |
| Group Operations | Membership queries, adding and removing members |
| Error Handling | LDAP result codes and how to branch on them |
| Computer Objects | Machine account queries and management |
| Testing | Integration tests against a directory with Testcontainers |

Read the relevant section rather than the whole file; each is self-contained.

## Rules that prevent the common failures

- **Never build a filter by string concatenation.** Escape every value that
  reaches a filter. An unescaped `)` or `*` in user input changes the query.
- **Bind failures are not all authentication failures.** Branch on the LDAP
  result code, not on the error message. A network fault, a referral and bad
  credentials are three different outcomes that a string match conflates.
- **Active Directory password writes require TLS** and a UTF-16LE encoded value
  wrapped in quotes for `unicodePwd`. A plaintext connection will fail the write
  rather than downgrade it.
- **Always close connections.** A leaked `*ldap.Conn` holds a server-side
  session, and directory servers cap concurrent connections per account.
- **Prefer paged search for anything unbounded.** Directory servers apply a size
  limit and will truncate a large result set silently.

## Related Skills

- **go-development** for the surrounding Go process, error handling idioms and
  the quality gate
