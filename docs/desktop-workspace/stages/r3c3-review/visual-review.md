# R3c3 native visual review

Expert walkthrough on macOS arm64, actual Electron source build, synthetic PDF content. This is
not representative-user research or a packaged cross-platform claim.

| User scenario                          | Checklist / observation                                                                                                                                                                             |
| -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Agent points while User composes       | Authored amber outline appears in PDF and Chat; draft, selection and navigation stay unchanged. No focus takeover.                                                                                  |
| User shares a region                   | Existing selection toolbar provides Share mark; no Agent turn or automatic Send is created.                                                                                                         |
| User requests Show on a different page | Temporary reader shows exact referenced page. Return remains explicit; base page, scale, rotation, scroll, selection and draft survive.                                                             |
| Source changes or disappears           | Show explains unavailability and does not apply a misleading highlight. Matching stored evidence opens its original PNG.                                                                            |
| Measured 900×650 content viewport      | Workspace/Chat switch retains state; Return and PDF controls stay accessible.                                                                                                                       |
| Pointer label overlaps source text     | Found during the real Agent trial. Corrected by moving the short author label outside the rectangle and retaining the full explanation in Chat. Final scripted native captures show the correction. |
| Same content, different actors         | User local selection remains teal; Agent mark is an authored amber outline. Images delivered to the Agent and stored captures contain source pixels, not UI overlays.                               |

The owner should review the final native screenshots and interaction before the next View family.
