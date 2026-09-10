## Artist query-index evidence

### Task 5.1 — candidate selection

Evidence was captured with SQLite 3.53.3 from a consistent backup of the local
production-shaped PocketBase database: 5,829 eligible published artists and
52,866 artworks. The exercised list shape selected `Artists.*`, applied the
canonical published/complete-identity predicate, and used `LIMIT 60 OFFSET 0`.
Only the `ORDER BY` changed:

- name ascending: `filing_name ASC, id ASC`
- name descending: `filing_name DESC, id ASC`
- birth: `(year_of_birth = 0) ASC, year_of_birth ASC, filing_name ASC, id ASC`

Before the candidates, all three shapes searched
`pbx_artist_published_name (published=?)` and reported
`USE TEMP B-TREE FOR ORDER BY`.

The selected candidates and resulting plans are:

| Index                                                                                              | Exercised result                                                                                                                                                                                                     |
| -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pbx_artist_published_filing` on `(published, filing_name, id)`                                    | Name ascending searches this index with no temporary sort. Name descending uses it but retains `USE TEMP B-TREE FOR LAST TERM OF ORDER BY` because the deterministic `id ASC` direction differs from a reverse scan. |
| `pbx_artist_published_filing_desc` on `(published, filing_name DESC, id ASC)`                      | Name descending searches this index with no temporary sort.                                                                                                                                                          |
| `pbx_artist_published_birth` on `(published, (year_of_birth = 0), year_of_birth, filing_name, id)` | Birth order searches this index with no temporary sort.                                                                                                                                                              |

A plain birth candidate on
`(published, year_of_birth, filing_name, id)` was rejected: the unknown-year
ordering expression still produced `USE TEMP B-TREE FOR ORDER BY`.

Database-size comparison used a compact backup (`VACUUM`, 4,096-byte pages) so
free pages could not hide index allocation:

| State                    |      Pages |           Bytes |                      Delta |
| ------------------------ | ---------: | --------------: | -------------------------: |
| Baseline                 |     45,255 |     185,364,480 |                          — |
| Ascending name added     |     45,319 |     185,626,624 |                   +262,144 |
| Descending name added    |     45,383 |     185,888,768 |                   +262,144 |
| Birth order added        |     45,454 |     186,179,584 |                   +290,816 |
| **All selected indexes** | **45,454** | **186,179,584** | **+815,104 bytes (0.44%)** |

The un-compacted working snapshot had 146 free pages; the first candidate reused
64 of them and therefore showed no physical file growth. The compact comparison
above records the actual allocation rather than that incidental file-size result.

## Artwork list/count experiment

### Task 6.1 — combined window count retained

The candidate selects the ordered page IDs and `COUNT(*) OVER()` in one filtered
query, then hydrates only those sixteen IDs. The previous implementation ran the
complete filter predicates once for `COUNT(DISTINCT id)` and again for the page.
An out-of-range page performs a fallback window read to recover the total and
continues to clamp to the canonical final page.

Deterministic equivalence passed for unfiltered, title/artist text, exact artist
(including a synthetic co-authored match), venue, and date-range cases. The same
five cases also matched total counts and ordered first-page IDs against the
production-shaped 52,866-artwork database. Existing handler tests continued to
pass the empty-result and out-of-range-page contracts.

Benchmark command:

```text
WGA_PERF_DATA_DIR=<isolated-production-shaped-data> go test ./internal/handlers/artworks \
  -run '^$' -bench '^BenchmarkArtworkListCountExperiment$' \
  -benchmem -benchtime=10x -count=3
```

Median results on an AMD Ryzen 7 7800X3D were:

| Case             | Separate ns/op | Combined ns/op |   Time | Bytes/op | Allocs/op |
| ---------------- | -------------: | -------------: | -----: | -------: | --------: |
| Unfiltered       |    206,875,197 |    153,419,884 | -25.8% |    -9.3% |     -4.3% |
| Text (`Madonna`) |    339,097,616 |    185,719,145 | -45.2% |   -15.5% |     -9.0% |
| Exact artist     |    199,656,008 |    127,544,144 | -36.1% |   -12.1% |     -6.9% |
| Venue            |     14,114,029 |     10,837,172 | -23.2% |    -8.4% |     -4.8% |
| Date range       |    190,994,341 |    111,192,226 | -41.8% |    -8.9% |     -5.4% |

A five-case, five-iteration process-level comparison reported 5.11 seconds of
user CPU for the separate path and 3.49 seconds for the combined path (-31.7%);
system CPU fell from 1.73 to 1.15 seconds. Every required case improved elapsed
time, bytes, and allocations, so the combined path was retained.
