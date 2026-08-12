# Indexing Benchmark Corpus

This controlled corpus backs the public ThingIndex indexing benchmark exposed by `GET /api/v1/meta/indexing-benchmark`.

It exists to keep the baseline explicit and reproducible:

- each document fixture is embedded in the Go binary;
- fixtures may live inline in `corpus.json` or as embedded files referenced by `content_file`;
- `history.json` records versioned benchmark snapshots so pipeline changes can be compared before/after with the same key metrics;
- `reference-workspace.json` keeps a normalized snapshot of `pageindex/results` so the public benchmark can compare ThingIndex against the workspace reference bar;
- the public report also compares the current relational retrieval path against explicit flat and hybrid experimental controls over the same corpus;
- the benchmark runs the real deterministic indexer over those fixtures;
- document-tree cases validate exact top-1 node/page matches;
- collection cases validate exact top-1 document matches;
- structure checks validate expected outline titles and table counts.
- structural signals also report expected query-node presence, useful outline depth, outline coverage, and unexpected heading rate.

The benchmark is intentionally small and deterministic. It is a runtime baseline, not a claim of general production accuracy for arbitrary corpora or grounded chat answers.
