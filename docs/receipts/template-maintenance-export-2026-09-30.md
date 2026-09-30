# Template maintenance and application export — 2026-09-30

The unpublished 1.9.0 source now keeps template-repository workflow assertions in `.github/scripts/check-template-maintenance.py`. Template CI runs this static check explicitly. Those assertions previously lived inside the application `TestTemplateContract`, causing applications exported without `.github/` to fail unrelated application checks. All remaining application/manifest/naming checks are preserved. No conditional skip based on missing CI files was added.

The earlier project-configuration loader migration already removed `internal/connectorconfiguration/configuration_test.go`. Its old world-readable broker-token negative check is not restored. Superverse V2's separately versioned export adaptation for the published1.8.0 source explicitly calls `os.Chmod(0644)` in that existing negative check, preserving its premise under restrictive umask.

Static checks passed: the actual repository maintenance script, template-version script (1.9.0 remains unpublished), Python in-memory compilation, Go compilation of `internal/templatecontract` without test execution, and whitespace checks. Source inspection confirmed application checks contain no `.github` references and retain the application manifest and concrete naming checks. No application/provider integration or E2E was executed for this maintenance change, and no dependency release or publication occurred.

This adds no Flow, Step, primitive, application database, or UI behavior. The previously prepared project-configuration work in this isolated tree remains a separate unpublished change; these static results do not revalidate that entire change.
