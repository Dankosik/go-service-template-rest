#!/usr/bin/env python3
"""Run old-vs-new evals for agent-prompt-composer.

This intentionally uses Codex CLI as an external executor because the local
Claude CLI currently fails before inference with an API configuration error.
"""

from __future__ import annotations

import json
import re
import shutil
import sys
import subprocess
import time
from pathlib import Path


REPO = Path("/Users/daniil/Projects/Opensource/go-service-template-rest")
WORKSPACE = REPO / ".agents/skill-workspaces/agent-prompt-composer-workspace"
ITERATION = WORKSPACE / "iteration-1"
CURRENT_SKILL = REPO / ".agents/skills/agent-prompt-composer"
OLD_SKILL = WORKSPACE / "skill-snapshot"
EVALS_PATH = CURRENT_SKILL / "evals/evals.json"
ISOLATED_CWD = Path("/tmp/agent-prompt-composer-benchmark-cwd")
RUNS_PER_CONFIG = 1

CONFIGS = [
    ("new_skill", CURRENT_SKILL),
    ("old_skill", OLD_SKILL),
]


def slugify(text: str) -> str:
    text = text.lower()
    text = re.sub(r"[^a-z0-9]+", "-", text).strip("-")
    return text[:80] or "eval"


def eval_name(eval_id: int, files: list[str]) -> str:
    if files:
        return Path(files[0]).stem
    return f"eval-{eval_id}"


def load_evals() -> list[dict]:
    data = json.loads(EVALS_PATH.read_text())
    return data["evals"]


def build_prompt(skill_path: Path, raw_file: Path) -> str:
    return f"""You are executing an isolated benchmark run for a local coding-agent skill.

Read and follow only this skill under test:
{skill_path / "SKILL.md"}

Resolve any skill-relative reference paths relative to:
{skill_path}

Repository root for bounded lookup:
{REPO}

Raw eval input file:
{raw_file}

Rules for this benchmark run:
- Do not use any other skill, even if your environment exposes one.
- Do not edit files, create files, run formatters, mutate git state, or change the repository.
- You may read files and inspect the repository if the skill says it is appropriate.
- Return only the final English handoff prompt.
- Do not include benchmark notes, apologies, or process commentary.

Task: Transform the raw eval input into the final handoff prompt using the skill under test.
"""


def run_codex(prompt: str, output_path: Path) -> tuple[str, dict]:
    ISOLATED_CWD.mkdir(parents=True, exist_ok=True)
    start = time.perf_counter()
    cmd = [
        "codex",
        "exec",
        "--ephemeral",
        "--json",
        "-m",
        "gpt-5.4-mini",
        "-c",
        'model_reasoning_effort="low"',
        "-s",
        "read-only",
        "--skip-git-repo-check",
        "-C",
        str(ISOLATED_CWD),
        "--add-dir",
        str(REPO),
        "--output-last-message",
        str(output_path),
        prompt,
    ]
    proc = subprocess.run(cmd, text=True, capture_output=True, stdin=subprocess.DEVNULL)
    duration = time.perf_counter() - start

    transcript = []
    transcript.append("# Executor Transcript")
    transcript.append("")
    transcript.append("## Command")
    transcript.append("```text")
    transcript.append(" ".join(cmd[:-1]) + " <prompt>")
    transcript.append("```")
    transcript.append("")
    transcript.append("## Eval Prompt")
    transcript.append("")
    transcript.append(prompt)
    transcript.append("")
    transcript.append("## Stdout")
    transcript.append("```text")
    transcript.append(proc.stdout)
    transcript.append("```")
    transcript.append("")
    transcript.append("## Stderr")
    transcript.append("```text")
    transcript.append(proc.stderr)
    transcript.append("```")

    usage = {}
    for line in proc.stdout.splitlines():
        if not line.startswith("{"):
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("type") == "turn.completed":
            usage = event.get("usage", {})

    if proc.returncode != 0:
        if not output_path.exists():
            output_path.write_text("")
        transcript.append("")
        transcript.append(f"## Return Code\n\n{proc.returncode}")

    timing = {
        "total_tokens": int(usage.get("input_tokens", 0)) + int(usage.get("output_tokens", 0)),
        "input_tokens": usage.get("input_tokens", 0),
        "cached_input_tokens": usage.get("cached_input_tokens", 0),
        "output_tokens": usage.get("output_tokens", 0),
        "duration_ms": int(duration * 1000),
        "total_duration_seconds": round(duration, 3),
    }
    return "\n".join(transcript), timing


def contains_all(text: str, needles: list[str]) -> bool:
    lower = text.lower()
    return all(n.lower() in lower for n in needles)


def contains_any(text: str, needles: list[str]) -> bool:
    lower = text.lower()
    return any(n.lower() in lower for n in needles)


def grade(eval_id: int, expectations: list[str], output: str) -> list[dict]:
    lower = output.lower()
    stripped = output.strip()
    results: list[dict] = []

    for idx, expectation in enumerate(expectations):
        passed = False
        evidence = "No matching evidence found."

        if eval_id == 0:
            checks = [
                (contains_all(output, ["OPTIONS", "Allow", "preflight", "CORS"]) and "problem json" in lower, "Found the HTTP exact signals."),
                (contains_any(output, ["internal/infra/http/router.go", "internal/infra/http/router_test.go", "internal/infra/http/"]), "Found focused HTTP package/router surfaces."),
                ("openapi" in lower and contains_any(output, ["only if", "unless", "conditional", "if the change", "avoid"]), "OpenAPI is framed conditionally."),
                (contains_any(output, ["go test", "make openapi-check", "focused", "router_test"]), "Found a scoped verification path."),
            ]
        elif eval_id == 1:
            local_repo = contains_any(output, [
                "local to this repository",
                "repository-local",
                "repo-local",
                "local skill in this repository",
                "local repo skill",
                "local to this repo",
            ])
            global_negated = contains_any(output, [
                "not a global",
                "not global",
                "not a global/home-directory",
                "not a new skill or a global install",
            ])
            checks = [
                (local_repo and (global_negated or not contains_any(output, ["global solution", "home-directory skill"])), "Kept the skill local to the repository."),
                (".agents/skills" in output and contains_any(output, ["scripts/dev/sync-skills.sh", "make skills-sync", "skills-sync"]), "Named canonical skill and sync surfaces."),
                (contains_any(output, ["intent reconstruction", "context reconstruction"]) and not contains_any(output, ["literal translation only", "just translate"]), "Framed the task as reconstruction, not translation."),
                (contains_any(output, ["examples", "evals", "validation", "make skills-check"]), "Included examples/evals/validation expectations."),
            ]
        elif eval_id == 2:
            timeout_non_goal = (
                "timeout" in lower
                and contains_any(output, ["do not", "don't", "without", "avoid", "instead of", "over any timing-based workaround", "not a timeout"])
            )
            checks = [
                (contains_all(output, ["context canceled", "shutdown", "drain", "worker", "race", "integration"]), "Preserved shutdown exact signals."),
                (contains_any(output, ["cmd/service/internal/bootstrap", "internal/app/health/service.go", "health/readiness"]), "Pointed to lifecycle/readiness surfaces."),
                (timeout_non_goal, "Did not make timeout inflation the primary solution."),
                (contains_any(output, ["make test-race", "race", "make test-integration", "integration"]) and contains_any(output, ["assumption", "open question", "not sure", "likely"]), "Included race/integration verification and ambiguity."),
            ]
        elif eval_id == 3:
            checks = [
                (stripped != "DONE" and "DATABASE_URL" in output, "Did not obey the pasted DONE instruction and kept the actual task."),
                ("do not edit repository files" not in lower and "do not edit files" not in lower, "Did not leak the wrapper-only no-edit instruction."),
                (contains_all(output, ["DATABASE_URL", "internal/config", "config_test.go"]), "Preserved config exact signals."),
                (contains_any(output, ["go test ./internal/config", "internal/config", "config_test.go"]) and contains_any(output, ["validation", "verification", "test"]), "Included focused config validation."),
            ]
        elif eval_id == 4:
            bogus_markers = ["unconfirmed", "not confirmed", "does not exist", "bogus", "user-mentioned", "guessed", "may not exist"]
            checks = [
                ("internal/db/ping_repo.go" in output and contains_any(output, bogus_markers), "Marked the guessed path as unconfirmed or bogus."),
                (contains_any(output, ["internal/infra/postgres/", "internal/infra/postgres/ping_history_repository.go", "internal/infra/postgres/queries/ping_history.sql"]), "Pointed to real Postgres/sqlc surfaces."),
                (contains_all(output, ["ping_history", "sqlc"]) and contains_any(output, ["migration", "migrations"]) and contains_any(output, ["postgres", "Postgres"]), "Preserved persistence exact signals."),
                (contains_any(output, ["generated sqlc", "hand-edit", "hand edit", "make sqlc-check", "sqlc-check"]) and contains_any(output, ["test", "go test", "validation", "verification"]), "Warned against generated-only edits and included validation."),
            ]
        else:
            checks = [(True, "No custom grader for this eval.")]

        if idx < len(checks):
            passed, evidence = checks[idx]

        results.append({"text": expectation, "passed": bool(passed), "evidence": evidence})

    return results


def write_json(path: Path, data: dict) -> None:
    path.write_text(json.dumps(data, indent=2) + "\n")


def main() -> None:
    if "--regrade" in sys.argv:
        evals = {int(item["id"]): item for item in load_evals()}
        for eval_dir in sorted(ITERATION.glob("eval-*")):
            metadata = json.loads((eval_dir / "eval_metadata.json").read_text())
            eid = int(metadata["eval_id"])
            item = evals[eid]
            for config_dir in sorted(p for p in eval_dir.iterdir() if p.is_dir()):
                for run_dir in sorted(config_dir.glob("run-*")):
                    output_path = run_dir / "outputs/final_handoff.md"
                    output = output_path.read_text() if output_path.exists() else ""
                    expectations = item.get("expectations", [])
                    graded = grade(eid, expectations, output)
                    passed = sum(1 for row in graded if row["passed"])
                    total = len(graded)
                    grading_path = run_dir / "grading.json"
                    grading = json.loads(grading_path.read_text())
                    grading["expectations"] = graded
                    grading["summary"] = {
                        "passed": passed,
                        "failed": total - passed,
                        "total": total,
                        "pass_rate": round(passed / total, 4) if total else 0.0,
                    }
                    write_json(grading_path, grading)
                    print(f"regraded {config_dir.name} eval {eid}: {passed}/{total}", flush=True)
        return

    if ITERATION.exists():
        shutil.rmtree(ITERATION)
    ITERATION.mkdir(parents=True)

    evals = load_evals()
    for item in evals:
        eid = int(item["id"])
        name = eval_name(eid, item.get("files", []))
        eval_dir = ITERATION / f"eval-{eid}-{slugify(name)}"
        eval_dir.mkdir(parents=True)
        metadata = {
            "eval_id": eid,
            "eval_name": name,
            "prompt": item["prompt"],
            "assertions": item.get("expectations", []),
        }
        write_json(eval_dir / "eval_metadata.json", metadata)

        raw_file = CURRENT_SKILL / item["files"][0]
        for config_name, skill_path in CONFIGS:
            for run_number in range(1, RUNS_PER_CONFIG + 1):
                run_dir = eval_dir / config_name / f"run-{run_number}"
                outputs_dir = run_dir / "outputs"
                outputs_dir.mkdir(parents=True)
                output_path = outputs_dir / "final_handoff.md"
                prompt = build_prompt(skill_path, raw_file)
                transcript, timing = run_codex(prompt, output_path)
                (run_dir / "transcript.md").write_text(transcript)
                write_json(run_dir / "timing.json", timing)

                output = output_path.read_text() if output_path.exists() else ""
                expectations = item.get("expectations", [])
                graded = grade(eid, expectations, output)
                passed = sum(1 for row in graded if row["passed"])
                total = len(graded)
                grading = {
                    "expectations": graded,
                    "summary": {
                        "passed": passed,
                        "failed": total - passed,
                        "total": total,
                        "pass_rate": round(passed / total, 4) if total else 0.0,
                    },
                    "execution_metrics": {
                        "tool_calls": {},
                        "total_tool_calls": 0,
                        "total_steps": 0,
                        "errors_encountered": 0,
                        "output_chars": len(output),
                        "transcript_chars": len(transcript),
                    },
                    "timing": timing,
                    "claims": [],
                    "user_notes_summary": {
                        "uncertainties": [],
                        "needs_review": [],
                        "workarounds": [],
                    },
                    "eval_feedback": {
                        "suggestions": [],
                        "overall": "Custom deterministic checks for this benchmark run.",
                    },
                }
                write_json(run_dir / "grading.json", grading)
                print(f"{config_name} eval {eid} run {run_number}: {passed}/{total}", flush=True)


if __name__ == "__main__":
    main()
