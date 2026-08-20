# wifimgr and the NAF Framework

I wrote `wifimgr` to manage Mist infrastructure from a CLI, because Mist's API was the best of the wireless vendors I got to work with. A tool like this only becomes useful when you can manage more than one vendor so it grew to cover Meraki, Aruba Instant, and Ubiquiti. That growth cost me a significant rewrite (or two): my first data model was Mist's data model, and it did not survive integration with the second vendor let alone the third.

Then I sat in the [NAF Framework track at AutoCon 5](https://networkautomation.forum/autocon5#program) and spent it recognizing my own tool in the [boxes](https://reference.networkautomation.forum/Framework/images/naf-automation-framework-v1-dark.png) on the slides. Checking the [reference framework](https://reference.networkautomation.forum/Framework/Framework/) afterward, expecting it to be a human hallucination, I found five of its six functional blocks already accounted for.

I assure you that was an accident, not a design goal, and the rewrite is why. This page records the mapping, including the parts where `wifimgr` does not clear the framework's bar, as best I believe to be true and correct.

## The six blocks

| Block             | NAF role                                                                                             | wifimgr today                                                                                                                           |
| ----------------- | ---------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| **Intent**        | Stores desired state in a structured, vendor-neutral form; idempotent; derived into vendor artifacts | The JSON site config. Vendor-neutral structs, schema-validated, with `mist:` / `meraki:` extension blocks for what does not generalize. |
| **Collector**     | Retrieves actual state through read interfaces; normalizes vendor data                               | `show …` plus the cache. Vendor responses normalize into the same types the config uses.                                                |
| **Observability** | Persists history, generates events on intent-vs-actual discrepancy                                   | `apply … diff` computes drift between intent and actual.                                                                                |
| **Orchestrator**  | Coordinates workflows, event-driven                                                                  | Not present.                                                                                                                            |
| **Executor**      | Applies changes, idempotent                                                                          | `apply` pushes the neutral config through vendor REST APIs. `managed_keys` bounds what gets written.                                    |
| **Presentation**  | User-facing read/write interface                                                                     | The CLI.                                                                                                                                |

Five of six.

## Why the mapping works

The same neutral type set flows through Intent, Collector, and Executor without a translation layer bolted on afterward. That is not craft. It is the scars and bruises from getting it wrong the first 3 times.

`wifimgr` started as a Mist-only tool, and choosing it on API quality quietly made Mist's data model the tool's data model. When I added a second vendor, I spent a long stretch shoehorning Meraki into structs that a different vendor had shaped, and every bug I hit traced back to a modeling decision I had never made.

The rewrite gave each vendor its own representation and an explicit mapping into a neutral one. Three things followed from that, and they are the three the framework cares about:

- **The config is the neutral representation.** `RadioConfig` describes radio intent once, and per-vendor translators derive the Mist artifact (`band_24`, `power`) and the Meraki one (`fiveGhzSettings`, `targetPower`) from it.
- **The Collector reads back into the same types.** Drift detection compares like against like instead of special-casing each vendor.
- **`managed_keys` gives the Executor an idempotency boundary.** Only declared fields get pushed, so re-applying unchanged intent is a no-op.

`wifimgr` is a vertically integrated Intent > Collector > Observability > Executor pipe with a CLI on top. NAF describes the same pipe horizontally, as separable services.

## Where its origin story comes through

I met Mist on the Corp WiFi Engineering team at Google, before Juniper acquired them. The API was novel at the time, and it was the first one that made deploying and managing APs look like it could be better than what the incumbent vendor offered rather than merely automatable. I tried to bring Mist with me to the companies I went to afterward, for that reason.

So when this tool needed a vocabulary, it reached for the one I had been thinking in for years.

The neutral schema is not neutral, and the borrow runs deeper than the naming. The [field mappings reference](field-mappings.md) names Mist nomenclature as the standard in its design-philosophy section, which covers the spelling. `internal/vendors/ap_config_types.go` shows where Mist also set the shape:

| Neutral type                           | What it borrows                                                                                     |
| -------------------------------------- | --------------------------------------------------------------------------------------------------- |
| `Location []float64` (`[lat, lng]`)    | Mist's container. Meraki returns `lat` and `lng` as separate scalars, decomposed on the way in.     |
| `Band5On24Radio *RadioBandConfig`      | `band_5_on_24_radio` is a Mist radio concept, sitting first-class on the vendor-agnostic type.      |
| `MapID`, `MapName`, `X`, `Y`, `Height` | Mist's floorplan model, promoted to neutral.                                                        |
| `DualBandConfig.RadioMode *int`        | Neutral name, vendor-specific value domain: Mist means 2.4>5 conversion, Meraki means a 5/6 toggle. |
| `Mist` / `Meraki` extension blocks     | What counts as "common" is bounded by what Mist already expresses.                                  |

Rename every field tomorrow and all five survive, which is what separates a vocabulary problem from this one. Shaping an abstraction around your best-behaved vendor is a compromise, and the test is not whether the names read sensibly to an operator of a vendor you have not integrated yet. It is whether the containers do: that operator pays for the mismatch in a decomposition step, every read and every write. Meraki already does.

## The gap: nothing else can read it

What `wifimgr` lacks is not function. Every block exists, but they only talk to each other in-process, so nothing outside the binary can read what they produce. Slotting `wifimgr` into a larger NAF stack, as an Executor behind someone else's Orchestrator or a Collector feeding someone else's Observability, means each boundary has to put out something documented, versioned, and machine-readable:

- **Intent.** The config is the way in, but nothing derives its JSON Schema from the structs. I maintain it by hand in three trees (`config/schemas/`, `internal/schemadefs/`, `docs/schemas/`), and they have already disagreed: `inventory-schema.json` describes the per-site armed allowlist in one and a flat device-type list in the other two, under the same filename. Calling it a "well-defined schema" today would be generous. It needs to be generated from the structs, or verified against them in CI, and reduced to one canonical artifact before another tool can safely write to it.
- **Collector output.** `show … json` is machine-facing already (no ANSI, jq-safe, HTML escaping off) but unversioned and unpublished. A versioned output schema turns it into a state feed.
- **Observability.** Drift is human-readable text. A structured per-field drift event from `diff` would turn it into a signal something else can consume.
- **Executor.** A non-interactive `apply` that reads intent from stdin and emits a per-device structured result would let an external orchestrator drive it.

None of these require re-architecture. Every one is a new way to read something the tool already does.

## Direction

The config comes first, because the other three describe the same types it defines. The Collector's JSON is those types, a drift event is a per-field comparison of them, and a non-interactive `apply` reads them on stdin. Version any of those before the schema is canonical and you version it twice. So: one schema, generated from or verified against the structs, and everything else versioned on top of it.

`wifimgr` does not need NAF features added. It needs to expose the boundaries it already has.
