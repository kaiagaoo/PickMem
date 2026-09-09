# AI engineering notes

PickMem treats long-term memory as two separate problems:

1. **Candidate generation:** which stored facts may be useful for this task?
2. **Disclosure:** which facts is the user willing to send to this model now?

Many memory systems collapse these into automatic retrieval and prompt
injection. PickMem keeps the second step explicit. A ranker can recommend, but
only the picker writes `pickmem/active.json`.

## Ranking pipeline

`pickmem suggest <task>` runs locally over active notes:

```text
task description
      |
      v
Unicode tokenization --> document frequencies --> field-weighted BM25
                                                   |
                                                   v
                                  score + matched-term explanation
                                                   |
                                                   v
                                     read-only suggestions
                                                   |
                                         user approves in picker
```

The scorer weights labels, tags, and groups more strongly than body text. IDF
reduces the influence of common terms, BM25 length normalization prevents long
notes from winning merely because they contain more words, and stable ID
tie-breaking keeps runs reproducible. This is intentionally a transparent
baseline: embeddings or a reranker can be evaluated later without changing the
human approval boundary.

## Offline evaluation

`pickmem eval` compares the field-weighted BM25 ranker with a token-overlap
baseline. It calculates:

- **Precision@k:** what fraction of the context budget is relevant.
- **Recall@k:** what fraction of labeled relevant memories was retrieved.
- **MRR:** how early the first relevant memory appears.
- **Context reduction:** how much of the vault was excluded from the candidate
  set.

The bundled `pickmem-synthetic-context-v1` dataset is small and synthetic. Its
purpose is deterministic regression testing and demonstrating the evaluation
contract—not supporting a real-world accuracy claim. A credible product study
should replace it with consented task/memory pairs, hold out the test set, and
report confidence intervals plus failure categories.

### Dataset format

```json
{
  "name": "my-evaluation-v1",
  "notes": [
    {
      "id": "editor",
      "label": "Editor preference",
      "group": "work/stack",
      "tags": ["preference"],
      "body": "Uses Neovim for daily development."
    }
  ],
  "cases": [
    {
      "query": "configure my development environment",
      "relevant_ids": ["editor"]
    }
  ]
}
```

Keep identifying or sensitive user data out of committed datasets. Private
datasets can be passed with `--dataset` and should remain ignored or stored in
an approved secure location.

## Memory extraction

The MCP server exposes `stage_memories` for structured candidates extracted by
the connected model. Candidates are deduplicated and routed into an inbox; they
do not become active memory until accepted. The experimental `prototype/`
harness explores model-based extraction with explicit `skipped`, `confidence`,
and `why` fields so false-positive decisions can be inspected.

An extraction evaluation should separately measure candidate precision,
durability, deduplication, consolidation, taxonomy accuracy, and sensitive-data
handling. Retrieval scores alone do not measure those behaviors.

## Threat model

PickMem assumes memory may contain sensitive information and captured text may
be adversarial.

- The web API accepts loopback hosts only and rejects cross-origin and
  cross-site browser requests, reducing localhost CSRF and DNS-rebinding risk.
- MCP uses stdio, so it is available only to the client process that launches
  it.
- Suggested context is never activated automatically.
- AI-extracted memories remain pending until human review.
- Writes are atomic, and edits refuse to overwrite a file that changed since
  indexing.

Remaining risks include prompt injection inside an approved memory, stale
active selections, malicious local processes, and users intentionally pasting
context into a remote provider. Future work should add provenance and
sensitivity metadata, session-scoped active selections, and instruction/data
boundaries in assembled context.
