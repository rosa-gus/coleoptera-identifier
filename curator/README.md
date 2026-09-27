# Photo Curator TUI

This is a separate Go application for preparing the backend evaluation data.
It does not call or modify the identification CLI or the identification API.
It uses Bubble Tea v2 and Lip Gloss v2, with `#66A3BF` as its accent color.

## Run

From the repository root:

```bash
cd curator
go run .
```

The tool finds `data/evaluation/manifest.csv` by walking up from the current
directory. Use `--data-dir /path/to/data/evaluation` to select another dataset
or `--observation 339647945` to prefill an observation ID. Go 1.25 or newer
and internet access to iNaturalist are required.

## Workflow

1. Enter an iNaturalist observation ID and press Enter. The app fetches the
   observation, taxonomic lineage, and available photo metadata. A blue loading
   indicator stays visible while each observation or photo request is running.
2. Use `[` and `]` to choose a photo. The app loads only the selected photo
   for a terminal preview. Each photo has its own ID, attribution, and license.
   Use the link below the preview or press `o` to open the selected photo's
   iNaturalist page in a browser. That page offers larger image sizes and the
   photo's attribution and license.
3. Review the fields in **Taxonomy**, **Credits**, **Verification**, and
   **File & Observation**. Use Left/Right to change sections, Up/Down to choose
   a field, and Enter to edit it. The immutable source and photo IDs are shown
   separately and cannot be changed by editing the CSV fields.
4. For a positive Coleoptera photo, independently verify that the species is
   accepted in the [CTFB](https://fauna.jbrj.gov.br/fauna/). Enter its taxon
   URL, the catalog publication/version, and the date checked. The API does
   not establish Brazilian catalog membership. The photo may have been taken
   outside Brazil.
5. Check the selected photo's actual attribution and license on its
   iNaturalist photo page. Review the terminal preview and metadata; correct
   any field returned incorrectly by the API. Press `t` to confirm taxonomy
   and `c` to confirm credits. These confirmations are required to save.
6. Press Ctrl+S to review the save summary, then Enter to save. The app writes
   the photo under the appropriate family or negative category, a neighboring
   `.credits.md` file, and one CSV row. The CSV `split` can remain empty until
   observations are divided into calibration and test sets.

For a deliberate non-Coleoptera negative, press `n` if the detected order was
incorrect, or leave the automatically selected negative status. Set a valid
`negative_category` (`other_insects`, `arachnids`, `plants`,
`other_organisms`, or `objects_no_organism`). CTFB fields are not required
for negatives. A photo identified only to genus cannot be saved as a positive
species example.

## Keys

| Key | Action |
| --- | --- |
| Left/Right or `h`/`l` | Change section. |
| Up/Down or `k`/`j` | Select field. |
| Enter | Edit field; accept an edit; confirm save. |
| Ctrl+V or terminal paste | Paste into the observation ID or an open field editor. |
| Esc | Cancel edit or return from save confirmation. |
| `[` / `]` | Select another photo from the observation. |
| `o` | Open the selected photo's iNaturalist page in a browser. |
| `t` / `c` | Confirm taxonomy / photo credits after manual review. |
| `n` | Toggle Coleoptera versus deliberate negative. |
| Ctrl+S | Open save confirmation when required fields are complete. |
| `r` | Start another observation after saving. |
| `q` | Quit when not editing. |

Duplicate observation/photo ID pairs are rejected even when a filename or
sample ID has been edited. Existing photos and credits are never overwritten.
For the dataset layout and field definitions, see
[the evaluation README](../data/evaluation/README.md).
