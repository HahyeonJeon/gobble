# FastQC report fixture

`fastqc-0.12.1.html` is the complete 537181-byte output of the owned synthetic 100-read Trim Galore → FastQC 0.12.1 qualification. Original source SHA-256: `8f045465fd56bbfb18360592e1b41c4978b45c6054fb6bc2cef52aa724413d52`. It has ten sections and eight original graphs. No research/patient data.

Source/run evidence is recorded in `docs/desktop-workspace/stages/report-output-evidence/live-final/actual-report.json`. FastQC source and its embedded decorative assets are GPL-3.0-or-later, copyright Simon Andrews / Babraham Institute: https://github.com/s-andrews/FastQC/tree/v0.12.1 . The reader profile freezes only this qualified stylesheet and known decorative icons; it never applies source CSS.

The actual emitted data URLs sometimes omit 1–2 final IEND CRC bytes. Tests retain those original bytes and require successful bitmap decoding; they do not repair/re-encode images.
