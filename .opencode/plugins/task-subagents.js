export const TaskSubagents = async () => {
  const catalog = [
    "acceptance-unit-lead: own one implementation unit through proof and review",
    "worker-agent: one bounded mutable implementation, investigation, or verification result",
    "specialist-agent: one named method on one bounded decision",
    "evidence-agent: bounded read-only evidence without gate authority",
    "reviewer-agent: independent review of one fixed candidate",
    "adjudicator-agent: one surviving material reviewer conflict",
  ].join("\n- ")

  return {
    "tool.definition": async (input, output) => {
      if (input.toolID !== "task") return
      output.description = [
        output.description,
        "",
        "This repository's Task subagent_type values (pass these even if built-ins are listed first):",
        `- ${catalog}`,
        "Do not use explore, general, or scout as a substitute for acceptance-unit-lead.",
      ].join("\n")
    },
  }
}
