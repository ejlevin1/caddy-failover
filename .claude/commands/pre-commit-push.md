# /pre-git-push [commit message]

Comprehensive pre-push validation ensuring code quality, test coverage, and documentation before pushing changes.

## Usage
```sh
/pre-git-push [optional commit message]
```

## Validation Process

### 1. Repository State Check

#### Step 1.1: Branch Protection Verification
**Detect protected branches** (main, master, develop, staging, release/*, production):
- Warn immediately if on protected branch
- Offer to create feature branch
- Require explicit confirmation to continue

#### Step 1.2: Status Summary
Show current branch, uncommitted changes, commits ahead, and protection status.

### 2. Code Formatting

#### Step 2.1: Format Changed Files Only
**IMPORTANT:** Only format files that have been modified in this commit.
```markdown
## 🎨 Formatting Modified Files

Detecting changed files...
Found [X] modified files to format

Applying formatters to YOUR changes only:
- [List of your modified files being formatted]

Files reformatted: [Count]
No other files touched.
```

**Never format files outside the current changeset** - this prevents:
- Unrelated formatting changes in PR
- Merge conflicts from formatting other code
- Accidental modification of working code
- Scope creep in commits

### 3. Test Execution

#### Step 3.1: Run Test Suites
Execute tests in order of importance:
- **Unit Tests** - Component isolation
- **Integration Tests** - Component interaction
- **E2E Tests** - User workflows
- **Performance Tests** - Response times and memory
- **Contract Tests** - API compatibility

#### Step 3.2: Results Summary
Show passed/failed/skipped counts, coverage metrics, and performance baselines.

### 4. Code Quality Checks

#### Step 4.1: Static Analysis
- Complexity analysis
- Duplication detection
- Dead code identification
- Type safety validation

#### Step 4.2: Security Scanning
Check for secrets, vulnerabilities, injection risks, and dependency issues.

### 5. Documentation Validation

#### Step 5.1: Impact Analysis
Identify what documentation needs updating based on changes:
- New features requiring docs
- API changes
- Configuration updates
- Breaking changes

#### Step 5.2: Documentation Checklist
- README.md current?
- API docs complete?
- CHANGELOG updated?
- Code comments adequate?

### 6. Build Verification

Ensure production build succeeds and check for warnings.

### 7. Custom Validations

Offer optional additional checks:
- Load testing
- Accessibility audit
- Database migration verification
- Browser compatibility

### 8. Final Review

#### Step 8.1: Summary
```markdown
## Validation Complete

✅ Passed: [List]
⚠️ Warnings: [List]
❌ Blockers: [List]

Quality Metrics:
- Coverage: XX%
- Complexity: X.X
- Build: Success

Ready to push? [Yes/No]
```

### 9. Git Operations

#### Step 9.1: Protected Branch Final Check
If on protected branch, require typed confirmation:
```
⚠️ Type "CONFIRM PUSH TO [BRANCH]" or we'll create a feature branch instead:
```

#### Step 9.2: Commit and Push
- Stage changes
- Create descriptive commit
- Push to remote
- Offer PR creation

## Best Practices

1. **Only format changed files** - Don't touch unrelated code
2. **Respect protected branches** - Use feature branches and PRs
3. **Test before pushing** - Catch issues locally
4. **Update docs immediately** - Harder to do later
5. **Check security always** - Vulnerabilities compound
6. **Monitor performance** - Prevent degradation

## Example Workflow

```sh
User: /pre-git-push fix: auth timeout

Claude: ⚠️ You're on 'main' branch (protected).
        Create feature branch instead? (recommended)

User: Yes

Claude: Created 'fix/auth-timeout' branch

        Formatting YOUR 3 changed files only...
        ✅ Tests: 145/145 passed
        ✅ Security: Clean
        📚 Update CHANGELOG.md for this fix?

User: Yes, updated

Claude: All validations passed!
        Pushing to origin/fix/auth-timeout...

        Create PR to main? [Yes/No]
```
