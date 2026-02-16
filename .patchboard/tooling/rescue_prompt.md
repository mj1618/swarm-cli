# Merge Bot Rescue Prompt Template
#
# Variables available (will be substituted):
#   {{PR_NUMBER}}   - The PR number
#   {{TITLE}}       - PR title
#   {{HEAD_REF}}    - Source branch name
#   {{BASE_REF}}    - Target branch name  
#   {{ISSUES}}      - Description of issues (e.g., "merge conflict", "failing tests")
#
# Sections marked with {{IF_MERGE_CONFLICT}} ... {{END_IF}} are included only for merge conflicts
# Sections marked with {{IF_FAILING_TESTS}} ... {{END_IF}} are included only for failing tests

I need you to rescue PR #{{PR_NUMBER}} ("{{TITLE}}").

Branch: {{HEAD_REF}} → {{BASE_REF}}
Issues: {{ISSUES}}

Please:
1. First checkout the PR: gh pr checkout {{PR_NUMBER}}
2. Fix the {{ISSUES}}
3. Commit and push your changes

{{IF_MERGE_CONFLICT}}
For merge conflicts:
- Pull latest from {{BASE_REF}}: git fetch origin {{BASE_REF}} && git merge origin/{{BASE_REF}}
- Resolve conflicts in the affected files
- NOTE: if the conflict is due to files or folders being added with the same name, you should rename or number files or folders in the blocked PR (treating main/master as authoritative) - this is particuarly relevant for patchboard tasks which use a T-XXXX naming standard
- git add the resolved files and commit
{{END_IF}}

{{IF_FAILING_TESTS}}
For failing tests:
- Run the tests locally to see failures
- Fix the code causing test failures
- Verify tests pass before committing

IMPORTANT for UI/integration test timeouts:
- Do NOT increase timeouts to make tests pass - this masks the real problem
- Timeouts usually indicate the expected element/state will NEVER appear, not that it's slow
- Look for root causes: API errors (4xx/5xx), missing data, incorrect selectors, state pollution from other tests
- Check server logs for errors during test execution
- If tests pass locally but fail on CI, suspect environmental state differences, not speed
{{END_IF}}

Once fixed, push the changes to update the PR.
