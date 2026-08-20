# wifimgr and the NAF Framework

I wrote `wifimgr` to manage Mist infrastructure from a CLI, because Mist's API was the best of the wireless vendors I got to work with. A tool like this only becomes useful when you can manage more than one vendor so it grew to cover Meraki, Aruba Instant, and Ubiquiti. That growth cost me a significant rewrite (or two): my first data model was really Mist's data model, and it did not survive integration with the second vendor let alone the third.

Then I sat in the [NAF Framework track at AutoCon 5](https://networkautomation.forum/autocon5#program) and spent it recognizing my own tool in the [boxes](https://reference.networkautomation.forum/Framework/images/naf-automation-framework-v1-dark.png) on the slides. Checking the [reference framework](https://reference.networkautomation.forum/Framework/Framework/) properly afterward, expecting it being a human hallucination, I found five of its six functional blocks already accounted for.

I assure you that was an accident, not a design goal, and the rewrite is why. This page records the mapping, including the parts where `wifimgr` does not clear the framework's bar as best that I believe to be true and correct.

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

## Why the mapping works so well

The same neutral type set flows through Intent, Collector, and Executor without a translation layer bolted on afterward. That is not craft. It is the scars and bruises from getting it wrong the first 3 times.

`wifimgr` started as a Mist-only tool, because I felt Mist's API was the best of the wireless vendors I had to work with. Choosing it that way quietly made Mist's data model the tool's data model. When I added a second vendor, I spent a long stretch shoehorning Meraki into structs that a different vendor had shaped, and every bug I hit traced back to a modeling decision I had never consciously made.

The rewrite gave each vendor its own representation and an explicit mapping into a neutral one. Three things followed from that, and they are the three the framework cares about:

- **The config is the neutral representation.** `RadioConfig` describes radio intent once, and per-vendor translators derive the Mist artifact (`band_24`, `power`) and the Meraki one (`fiveGhzSettings`, `targetPower`) from it.
- **The Collector reads back into the same types.** Drift detection compares like against like instead of special-casing each vendor.
- **`managed_keys` gives the Executor an idempotency boundary.** Only declared fields get pushed, so re-applying unchanged intent is a no-op.

`wifimgr` is a vertically integrated Intent > Collector > Observability > Executor pipe with a CLI on top. NAF describes the same pipe horizontally, as separable services.

## Where its origin story comes through

I met Mist on the Corp WiFi Engineering team at Google, before Juniper acquired them. The API was novel at the time, and it was the first one that made deploying and managing APs look like it could be better than what the incumbent vendor offered rather than merely automatable. I tried to bring Mist with me to the companies I went to afterward, for that reason.

So when this tool needed a vocabulary, it reached for the one I had been thinking in for years.

The neutral schema is not fully neutral. It borrows Mist's vocabulary, because Mist's terms sat closest to vendor-agnostic of the options in front of me. The [field mappings reference](field-mappings.md) states this in its first paragraph rather than hiding it.

Naming an abstraction after your best-behaved vendor is a real compromise, and the honest test is whether the names still read sensibly to an operator of a vendor you have not integrated yet. Some of wifimgr's do not.

## The gap: nothing else can read it

What `wifimgr` lacks is not function. Every block exists, but they only talk to each other in-process, so nothing outside the binary can read what they produce. Slotting `wifimgr` into a larger NAF stack, as an Executor behind someone else's Orchestrator or a Collector feeding someone else's Observability, means each boundary has to put out something documented, versioned, and machine-readable:

- **Intent.** The config is the way in, but its JSON Schema is hand-maintained, duplicated across several trees, and not derived from the structs. The copies have already drifted. Calling it a "well-defined schema" today would be generous. It needs to be generated from the structs, or verified against them in CI, and reduced to one canonical artifact before another tool can safely write to it.
- **Collector output.** `show … json` is machine-facing already (no ANSI, jq-safe, HTML escaping off) but unversioned and unpublished. A versioned output schema turns it into a state feed.
- **Observability.** Drift is human-readable text. A structured per-field drift event from `diff` would turn it into a signal something else can consume.
- **Executor.** A non-interactive `apply` that reads intent from stdin and emits a per-device structured result would let an external orchestrator drive it.

None of these require re-architecture. Every one is a new way to read something the tool already does.

## Direction

The config comes first, because the other three describe the same types it defines. The Collector's JSON is those types, a drift event is a per-field comparison of them, and a non-interactive `apply` reads them on stdin. Version any of those before the schema is canonical and you version it twice. So: one schema, generated from or verified against the structs, and everything else versioned on top of it.

The framing worth keeping: `wifimgr` is not a tool that needs NAF features added. It is a NAF pipe that needs its internal boundaries exposed.
