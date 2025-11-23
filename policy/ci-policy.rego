# policy/ci-policy.rego
package securebus

# Default deny — you must earn the right to run
default allow := false

# Only main branch and tags can run chaos/fuzz/adversarial
allow {
    input.branch == "main"
}

allow {
    startswith(input.branch, "refs/tags/")
}

# Critical: chaos, fuzz, adversarial MUST depend on opa_gate
allow {
    some job
    input.jobs[job].name == "chaos"
    "opa_gate" == input.jobs[job].needs[_]
}

allow {
    some job
    input.jobs[job].name == "fuzz_nightly"
    "opa_gate" == input.jobs[job].needs[_]
}

allow {
    some job
    input.jobs[job].name == "adversarial"
    "opa_gate" == input.jobs[job].needs[_]
}

# Block if any critical vuln found
deny_vulns["high_or_critical_vulnerabilities_found"] {
    some finding
    finding := input.findings[_]
    finding.severity == "HIGH"
}

deny_vulns["high_or_critical_vulnerabilities_found"] {
    some finding
    finding := input.findings[_]
    finding.severity == "CRITICAL"
}
