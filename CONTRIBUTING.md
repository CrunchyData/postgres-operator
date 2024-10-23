The Postgres Operator is an open source project primarily authored and
maintained by the team at Crunchy Data. All contributions are welcome: the
Postgres Operator uses the Apache 2.0 license and does not require any
contributor agreement to submit patches.

Our contributors try to follow good software development practices to help
ensure that the code that we ship to our users is stable. If you wish to
contribute to the Postgres Operator, be it code or documentation, please follow
the guidelines below.

Thanks! We look forward to your contribution.

# General Contributing Guidelines

All ongoing development for an upcoming release gets committed to the
**`main`** branch. The `main` branch technically serves as the "development"
branch as well, but all code that is committed to the `main` branch should be
considered _stable_, even if it is part of an ongoing release cycle.

Ensure any changes are clear and well-documented:

- If the changes include code, ensure all additional code has corresponding
documentation in and around it. This includes documenting the definition of
functions, statements in code, sections.

- The most helpful code comments explain why, establish context, or efficiently
summarize how. Avoid simply repeating details from declarations,. When in doubt,
favor overexplaining to underexplaining.

- Code comments should be consistent with their language conventions. For
example, please use `gofmt` [conventions](https://go.dev/doc/comment) for Go source code.

- Do not submit commented-out code. If the code does not need to be used
anymore, please remove it.

- While `TODO` comments are frowned upon, every now and then it is ok to put a
`TODO` to note that a particular section of code needs to be worked on in the
future. However, it is also known that "TODOs" often do not get worked on, and
as such, it is more likely you will be asked to complete the TODO at the time
you submit it.

- Write clear, descriptive commit messages. A guide for this is featured later
on in the documentation.

Please provide unit tests with your code if possible. If you are unable to
provide a unit test, please provide an explanation as to why in your pull
request, including a description of the steps used to manually verify the
changes.

Ensure your commits are atomic. Each commit tells a story about what changes
are being made. This makes it easier to identify when a bug is introduced into
the codebase, and as such makes it easier to fix.

All commits must either be rebased in atomic order or squashed (if the squashed
commit is considered atomic). Merge commits are not accepted. All conflicts must
be resolved prior to pushing changes.

**All pull requests should be made from the `main` branch.**

# Commit Messages

Commit messages should be as descriptive and should follow the general format:

```
A one-sentence summary of what the commit is.

Further details of the commit messages go in here. Try to be as descriptive of
possible as to what the changes are. Good things to include:

- What the changes is.
- Why the change was made.
- What to expect now that the change is in place.
- Any advice that can be helpful if someone needs to review this commit and
understand.
```

If you wish to tag a GitHub issue or another project management tracker, please
do so at the bottom of the commit message, and make it clearly labeled like so:

```
Issue: CrunchyData/postgres-operator#123
```

# Submitting Pull Requests

All work should be made in your own repository fork. When you believe your work
is ready to be committed, please follow the guidance below for creating a pull
request.

## Upcoming Features

Ongoing work for new features should occur in branches off of the `main`
branch.

## Unsupported Branches

When a release branch is no longer supported, it will be renamed following the
pattern `REL_X_Y_FINAL` with the key suffix being _FINAL_. For example,
`REL_3_2_FINAL` indicates that the 3.2 release is no longer supported.

Nothing should ever be pushed to a `REL_X_Y_FINAL` branch once `FINAL` is on
the branch name.

# Testing

We greatly appreciate any and all testing for the project.
There are several ways to help with the testing effort:

- Manual testing: testing particular features with a series of manual commands
or custom scripts
- Writing unit tests: testing specific sections of the code
- Writing integration tests: automatically testing scenarios that require a
defined series of steps, such as end-to-end tests
- Environmental & workload testing: testing the code against specific workloads,
deployment platforms, deployment models, etc.
