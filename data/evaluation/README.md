# Backend evaluation images

Place photos in `images/coleoptera/<Family>/` or one of the
`images/not_coleoptera/` directories. Create another family directory when
needed. The eight existing families have at least one accepted species in the
Brazilian catalog, as verified below. Their frequency among eligible photo
observations has not been measured.

The prototype's intended positive scope is Coleoptera species cataloged in
Brazil. This is a dataset selection rule, not yet a filter in the API. Include
a positive photo in the scored dataset only after its species name has been
checked against a recorded version of the [Catálogo Taxonômico
da Fauna do Brasil (CTFB)](https://fauna.jbrj.gov.br/fauna/). The catalog
establishes scope; the photo's observation country is recorded separately.
A family directory alone does not establish that a particular species is in
scope. Keep photos identified only to family or genus out of the scored set
until a species-level reference can be verified. Negative photos in
`images/not_coleoptera/` do not need a Brazilian catalog match.

## Family verification

The table lists one accepted species per family in the CTFB checklist published
through GBIF on 2026-08-01. Checked on 2026-09-27. These examples verify the
family directories; they do not preselect any images or establish the frequency
of photo observations. Check each future photo's species separately.

| Family | Example accepted species in the Brazilian catalog |
| --- | --- |
| Coccinellidae | [Neocalvia dentatofasciata](https://api.gbif.org/v1/species/240171674) |
| Scarabaeidae | [Trichillum arrowi](https://api.gbif.org/v1/species/244921966) |
| Chrysomelidae | [Microctenochira circumcincta](https://api.gbif.org/v1/species/244927394) |
| Curculionidae | [Xyleborus subductus](https://api.gbif.org/v1/species/253813491) |
| Sphindidae | [Eurysphindus brasiliensis](https://api.gbif.org/v1/species/240199363) |
| Hydroscaphidae | [Yara vanini](https://api.gbif.org/v1/species/244930082) |
| Georissidae | [Georissus humeralis](https://api.gbif.org/v1/species/240223099) |
| Clambidae | [Clambus falx](https://api.gbif.org/v1/species/240197791) |

## Finding and recording photos

The separate [Photo Curator TUI](../../curator/README.md) can fetch iNaturalist
metadata by observation ID, preview and select a photo, and prepare the CSV row.
Its fields remain editable, and saving requires manual review of taxonomy and
photo credits. The process below also works without the tool.

1. Search by **family** in [iNaturalist observations](https://www.inaturalist.org/observations),
   with **Photos** and **Research Grade** selected. Start with **all locations**;
   do not require the photo to have been taken in Brazil. For example, use this
   [Coccinellidae search](https://www.inaturalist.org/observations?taxon_name=Coccinellidae&photos=true&quality_grade=research&place_id=any).
   Browse the observations for species with usable photos, then check whether
   those species are accepted in the CTFB. The species in the verification
   table above prove catalog presence only; some may have few or no suitable
   photos. A Brazil location filter is optional for a separate geographic
   analysis, not a requirement for inclusion. See the [iNaturalist search guide](https://help.inaturalist.org/en/support/solutions/articles/151000169670-how-to-search-inaturalist-observations)
   and [search URL guide](https://help.inaturalist.org/en/support/solutions/articles/151000170738).
2. Open an observation and confirm that its identification is at species
   level. Research Grade alone is insufficient: some such observations are
   identified only to genus. Compare the scientific name with an **accepted
   species** in the CTFB and record the taxon URL, catalog publication or
   version, and check date. If the name differs, resolve the synonym before
   including the photo. See the [iNaturalist quality criteria](https://help.inaturalist.org/en/support/solutions/articles/151000169936).
3. Inspect the image itself. Exclude duplicates, images that do not show the
   labeled organism, and images whose identification cannot be checked. If
   several photos come from one observation, keep their observation ID and
   assign them to the same evaluation split.
4. Check the **photo's** creator and license before saving it; the observation
   data can have a different license. On iNaturalist, click the **i** icon
   beneath the selected photo, review its attribution and license, select a
   size, and save it manually. Do not add an all-rights-reserved photo without
   permission. See the [download instructions](https://help.inaturalist.org/en/support/solutions/articles/151000233085-how-to-download-a-photo-from-inaturalist)
   and [media license guidance](https://help.inaturalist.org/en/support/solutions/articles/151000169918-can-i-use-the-photos-and-sounds-that-are-posted-on-inaturalist-).
5. Save the image under `images/coleoptera/<Family>/` with a unique filename.
   Add a row to `manifest.csv` containing its relative path, the observation
   URL and ID, photo attribution and license, verified taxonomy, CTFB record,
   and review details. Leave `split` unset until observations have been grouped
   and divided into calibration and test sets.

For families with few iNaturalist photos, try the [GBIF occurrence gallery](https://www.gbif.org/occurrence/gallery).
Search by family first, then check the species of promising records against
the CTFB. Inspect the occurrence and its original publisher, and
check the image license separately from the occurrence license. GBIF may
display the same iNaturalist observation, so compare source IDs before adding
it. GBIF [describes its image gallery](https://www.gbif.org/news/3IGy7GZXdCocycuS2082A6/top-tip-occurrence-search-gallery-view)
and [multimedia licensing](https://techdocs.gbif.org/en/data-publishing/multimedia-publishing).
If no verifiable photo is available for a species, try another CTFB species
from the same family. If the family still has too few eligible photos, record
that gap and leave its directory empty for now. Do not relabel an unidentified
family-level photo as a species.

For deliberate negatives, collect photos of other insects, arachnids, plants,
other organisms, and objects or scenes without an organism. Put them in the
corresponding `images/not_coleoptera/` directory and record their source,
creator, license, and review details in the same manifest. Mark
`is_coleoptera=false` and `label_rank=outside_scope`.

## Manifest

`manifest.csv` contains only a header until images are added. Add one row per
photo. `image_path` is relative to this directory (for example,
`images/coleoptera/Coccinellidae/photo.jpg`).

| Field | Expected value |
| --- | --- |
| `sample_id` | Unique local photo identifier. |
| `image_path` | Relative photo path. |
| `split` | `calibration` or `test`; photos from the same observation stay in the same split. |
| `is_coleoptera` | `true` or `false`. |
| `family`, `genus`, `species` | Confirmed taxonomy. A positive scored example requires a verified species; leave these fields empty for negatives when unknown. |
| `label_rank` | `species` for in-scope scored examples; `outside_scope` for negatives. |
| `brazil_catalog_source` | Name of the Brazilian checklist used to verify the species, normally `CTFB`. |
| `brazil_catalog_taxon_url` | Link to the species record in that checklist. |
| `brazil_catalog_version`, `brazil_catalog_checked_at` | Checklist version and verification date (`YYYY-MM-DD`). |
| `model_catalog_presence` | `present`, `absent`, or `unknown` for the taxon in the evaluated model catalog. |
| `observation_id` | Source observation ID, used to detect duplicates. |
| `photo_id` | Source photo ID; together with `observation_id`, prevents duplicate photos even if filenames change. |
| `source`, `source_record_url` | Source name and observation page. |
| `observed_country` | Country where the photo observation was made, if known; this does not replace the species catalog check. |
| `photographer`, `image_license` | Photo creator and license, checked separately from the observation data. |
| `observed_at`, `accessed_at` | Observation and access dates in `YYYY-MM-DD` format, when available. |
| `reviewer`, `notes` | Label reviewer and any caveats. |

The directories contain `.gitkeep` files solely to retain the empty structure
in Git.
