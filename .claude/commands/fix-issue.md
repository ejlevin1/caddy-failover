# /fix-issue [problem description]

Systematically analyze and fix issues in the codebase with comprehensive planning, impact analysis, and testing strategies.

## Usage

```sh
/fix-issue [brief description of the problem]
```

## Issue Resolution Process

### 1. Branch Setup and Preparation

#### Step 1.1: Identify Working Branch
Ask the user: "Which branch should I base this fix on? (e.g., main, develop, or a specific feature branch)"

#### Step 1.2: Update Local Repository
```sh
# Fetch latest changes from remote
git fetch origin [branch-name]

# Checkout and update the base branch
git checkout [branch-name]
git pull origin [branch-name]

# Create new fix branch
git checkout -b fix/[descriptive-issue-name]
```

### 2. Problem Analysis Phase

#### Step 2.1: Clarify the Problem
Before proceeding, ensure complete understanding:
- **What is the expected behavior?**
- **What is the actual behavior?**
- **When did this issue start occurring?**
- **Are there specific conditions that trigger it?**
- **Are there error messages or logs?**
- **What has already been tried?**

If any aspect is unclear, ask specific questions before continuing.

#### Step 2.2: Deep Root Cause Analysis
**REQUIRED:** Perform comprehensive analysis including:

##### Search for Related Code
```sh
# Search for error messages
grep -r "error message" --include="*.go" --include="*.js" --include="*.ts"

# Search for related function names
grep -r "function_name" --include="*.go"

# Find all usages of the problematic component
grep -r "ComponentName" --include="*.go"
```

##### Trace Execution Flow
- Identify entry points
- Map the call chain
- Locate where the issue manifests
- Find where the root cause originates

##### Analyze Dependencies
```sh
# Check import statements
grep -r "import.*problematic_package"

# Review module dependencies
cat go.mod | grep relevant_package

# Check for recent changes
git log -p --since="2 weeks ago" -- path/to/problematic/file
```

##### Review Related Tests
```sh
# Find existing tests
find . -name "*_test.go" -exec grep -l "TestFunctionName" {} \;

# Check test coverage
go test -cover ./...
```

### 3. Solution Design Phase

#### Step 3.1: Generate Multiple Approaches
**REQUIRED:** Present 2-3 distinct high-level approaches:

##### Approach Format:
```markdown
## Approach 1: [Descriptive Name]
**Strategy:** [Brief description of the approach]
**Pros:**
- [Advantage 1]
- [Advantage 2]
**Cons:**
- [Disadvantage 1]
- [Disadvantage 2]
**Estimated Complexity:** [Low/Medium/High]
**Risk Assessment:** [Low/Medium/High]

## Approach 2: [Descriptive Name]
[Same structure as above]

## Approach 3: [Descriptive Name]
[Same structure as above]
```

#### Step 3.2: Get User Confirmation
Present approaches and ask: "Which approach would you prefer, or would you like me to explore alternative solutions?"

### 4. Detailed Implementation Planning

#### Step 4.1: Comprehensive Code Analysis
**REQUIRED:** After approach selection, perform deep analysis:

##### Pattern Recognition
```sh
# Identify coding patterns in the codebase
# Look for similar implementations
find . -name "*.go" -exec grep -l "similar_pattern" {} \;

# Review architectural patterns
grep -r "interface.*Handler" --include="*.go"
```

##### Impact Analysis
**REQUIRED:** Identify all affected areas:
```sh
# Find all files importing the module
grep -r "import.*module_name" --include="*.go"

# Find all references to the function/type
grep -r "\bFunctionName\b" --include="*.go"

# Check for interface implementations
grep -r "type.*struct" --include="*.go" | xargs grep -l "InterfaceName"
```

##### Documentation Research
**REQUIRED:** Use available resources:
```markdown
## Research Checklist:
- [ ] Use context7 MCP to lookup latest library documentation
- [ ] Search for similar issues in GitHub issues
- [ ] Check Stack Overflow for related problems
- [ ] Review library changelogs for breaking changes
- [ ] Consult official documentation for best practices
```

Example context7 usage:
- "Get documentation for [library-name] [specific-feature]"
- "Search for [error-message] in [library-name] docs"

#### Step 4.2: Identify Code Patterns to Follow
```sh
# Find established patterns for similar functionality
grep -r "func.*Similar" --include="*.go" -A 10 -B 2

# Review error handling patterns
grep -r "if err != nil" --include="*.go" -A 3

# Check logging patterns
grep -r "log\." --include="*.go"

# Review testing patterns
find . -name "*_test.go" -exec head -50 {} \; | grep -A 10 "func Test"
```

### 5. Implementation Plan Presentation

#### Step 5.1: Comprehensive Plan Document
**REQUIRED:** Present the complete plan before implementation:

```markdown
# Implementation Plan for [Issue Description]

## 1. Root Cause
[Detailed explanation of why the issue occurs]

## 2. Solution Overview
[High-level description of the chosen approach]

## 3. Code Changes

### File: path/to/file1.go
**Changes Required:**
- Line XX-YY: [Description of change]
- Line ZZ: [Description of change]
**Reason:** [Why this change is necessary]

### File: path/to/file2.go
[Same structure as above]

## 4. Potential Side Effects
- **Area 1:** [Description of potential impact]
  - Mitigation: [How to handle it]
- **Area 2:** [Description of potential impact]
  - Mitigation: [How to handle it]

## 5. Testing Strategy

### Unit Tests
- [ ] Test case 1: [Description]
- [ ] Test case 2: [Description]
- [ ] Edge case: [Description]

### Integration Tests
- [ ] Scenario 1: [Description]
- [ ] Scenario 2: [Description]

### Manual Testing Steps
1. [Step 1 description]
2. [Step 2 description]
3. [Expected result]

## 6. Rollback Plan
If issues arise after implementation:
1. [Rollback step 1]
2. [Rollback step 2]

## 7. Documentation Updates
- [ ] Update README if applicable
- [ ] Update inline code comments
- [ ] Update API documentation
- [ ] Add changelog entry
```

#### Step 5.2: Get Final Approval
Ask: "Does this implementation plan look good? Should I proceed with the fix, or would you like any modifications?"

### 6. Implementation Phase

#### Step 6.1: Implement Changes
- Follow the approved plan exactly
- Make incremental commits with clear messages
- Add comprehensive comments for complex logic
- Ensure code follows established patterns

#### Step 6.2: Testing Implementation
```sh
# Run existing tests
go test ./...

# Run specific test file
go test -v path/to/file_test.go

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...
```

#### Step 6.3: Write New Tests
```go
// Example test structure
func TestFixedFunctionality(t *testing.T) {
    // Arrange
    [setup code]

    // Act
    [execute function]

    // Assert
    [verify results]
}
```

### 7. Verification and Documentation

#### Step 7.1: Final Verification Checklist
- [ ] All tests pass
- [ ] No linting errors
- [ ] Code follows project conventions
- [ ] Comments added where necessary
- [ ] No debug code left behind
- [ ] Performance impact assessed
- [ ] Security implications reviewed

#### Step 7.2: Create Pull Request
```sh
# Commit all changes
git add -A
git commit -m "fix: [concise description of fix]

- [Bullet point 1 describing what was fixed]
- [Bullet point 2 describing how it was fixed]
- [Any additional relevant information]

Fixes #[issue-number]"

# Push branch
git push -u origin fix/[descriptive-issue-name]

# Create PR
gh pr create --title "fix: [issue description]" \
  --body "[Detailed PR description including problem, solution, and testing]"
```

## Required Analysis Tools

### Code Search and Analysis
- Use Grep tool for pattern matching
- Use Glob tool for file discovery
- Use Task tool for complex searches
- Use Read tool for examining specific files

### Documentation and Research
- Use context7 MCP for library documentation
- Use WebSearch for finding solutions
- Use WebFetch for reading documentation pages
- Check GitHub issues for similar problems

### Testing and Validation
- Run test suites
- Perform manual testing
- Verify edge cases
- Check performance impact

## Best Practices

1. **Never skip the analysis phase** - Understanding the problem fully prevents incorrect fixes
2. **Always present multiple approaches** - Different solutions have different trade-offs
3. **Document everything** - Future developers need to understand why changes were made
4. **Test comprehensively** - Automated and manual testing catch different issues
5. **Follow existing patterns** - Consistency makes code maintainable
6. **Consider side effects** - Every change can impact other parts of the system
7. **Plan for rollback** - Always have a way to undo changes if needed

## Common Pitfalls to Avoid

- **Fixing symptoms instead of root causes** - Dig deeper to find the real issue
- **Making assumptions without verification** - Always confirm your understanding
- **Ignoring edge cases** - These often cause the most problems
- **Skipping tests** - Tests prevent regressions
- **Not considering performance** - Fixes shouldn't degrade performance
- **Forgetting documentation** - Code without context is hard to maintain

## Output Format

The command should produce:
1. Clear problem analysis
2. Multiple solution approaches
3. Detailed implementation plan
4. Comprehensive test coverage
5. Complete documentation
6. Ready-to-merge pull request

## Example Workflow

```sh
User: /fix-issue The API returns 500 errors when processing large files

Claude: Which branch should I base this fix on? (e.g., main, develop, or a specific feature branch)

User: main

Claude: [Fetches latest from main, creates fix branch]
        [Performs deep analysis of the issue]
        [Presents 3 approaches with pros/cons]
        [After user selects approach, performs impact analysis]
        [Presents detailed implementation plan]
        [After approval, implements fix with tests]
        [Creates PR with comprehensive description]
```
