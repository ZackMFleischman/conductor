# Criterion-precondition behavioral evaluation

Review `criterion-precondition-packet-a.json` and `criterion-precondition-packet-b.json` using the current conductor-plan, conductor-work, conductor-worker, and workflow-command guidance. Decide whether each packet supplies evidence for every precondition in its criterion. Treat packet claims as claims to check against the packet's fact-to-artifact evidence map. Every citation must name the artifact that supports its exact attributed fact; split facts across citations when their sources differ. Do not infer directory ownership or process identity from a path, a checkout name, a successful command, or a named pass check.

Return one JSON object with exactly this shared response schema:

```json
{
  "reviews": [
    {
      "packet_id": "packet id",
      "criterion_status": "complete or incomplete",
      "evidence_scope": "historical-decision or later-prospective-addendum",
      "reason": "criterion-specific explanation",
      "citations": [
        {
          "artifact": "exact artifact path from the packet",
          "fact": "the fact this artifact establishes or fails to establish"
        }
      ]
    }
  ],
  "history": {
    "original_acceptance_rewritten": "<true or false>",
    "reason": "effect of later evidence on the historical decision"
  },
  "limits": "what this evidence and the existing structural validation checks cannot establish"
}
```

Include one review for each packet and replace `original_acceptance_rewritten` with a JSON Boolean. Base `criterion_status` on the actual criterion and cited evidence rather than headings, field presence, or the historical named pass flags. Do not run the application, tracker lifecycle, or startup commands.
