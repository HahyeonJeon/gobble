# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: notebook.spec.ts >> native text drag uses exact display offsets and rejects stale or foreign image requests
- Location: desktop/tests/electron/notebook.spec.ts:161:1

# Error details

```
Error: locator.evaluate: IndexSizeError: Failed to execute 'setStart' on 'Range': There is no child at offset 8.
    at eval (eval at evaluate (:311:30), <anonymous>:7:7)
    at UtilityScript.evaluate (<anonymous>:313:16)
    at UtilityScript.<anonymous> (<anonymous>:1:44)
```

# Page snapshot

```yaml
- generic [ref=e3]:
  - link "Skip to workspace" [ref=e4] [cursor=pointer]:
    - /url: "#workspace-main"
  - banner [ref=e5]:
    - button "Toggle files" [expanded] [ref=e6] [cursor=pointer]
    - heading "Notebook research" [level=1] [ref=e9]:
      - button "Switch Project" [ref=e10] [cursor=pointer]:
        - generic [ref=e11]: Notebook research
    - generic [ref=e14]:
      - generic "Workspace layout" [ref=e15]:
        - button "Single pane" [pressed] [ref=e16] [cursor=pointer]
        - button "Two panes" [ref=e19] [cursor=pointer]
      - button "Project selections" [ref=e22] [cursor=pointer]:
        - text: Selections
        - generic [ref=e23]: "0"
      - button "Account" [ref=e25] [cursor=pointer]
  - generic [ref=e27]:
    - complementary "Project navigation" [ref=e28]:
      - generic [ref=e29]:
        - group [ref=e30]:
          - generic "› Files" [ref=e31] [cursor=pointer]
          - generic [ref=e34]:
            - button "Project files" [disabled] [ref=e35]
            - button "Refresh files" [ref=e36] [cursor=pointer]
          - list [ref=e39]:
            - listitem [ref=e40]:
              - button "analysis.ipynb" [active] [ref=e41] [cursor=pointer]
              - button "Open analysis.ipynb in the other pane" [ref=e45] [cursor=pointer]
            - listitem [ref=e48]:
              - button "notes.txt" [ref=e49] [cursor=pointer]
              - button "Open notes.txt in the other pane" [ref=e53] [cursor=pointer]
            - listitem [ref=e56]:
              - button "quality.png" [ref=e57] [cursor=pointer]
              - button "Open quality.png in the other pane" [ref=e61] [cursor=pointer]
            - listitem [ref=e64]:
              - button "samples.csv" [ref=e65] [cursor=pointer]
              - button "Open samples.csv in the other pane" [ref=e69] [cursor=pointer]
        - group [ref=e72]:
          - generic "› Runs" [ref=e73] [cursor=pointer]
          - generic [ref=e76]:
            - generic [ref=e77]: Existing analyses
            - button "Refresh runs" [ref=e78] [cursor=pointer]
          - list [ref=e81]
          - paragraph [ref=e82]: No runs attached
    - generic [ref=e83]:
      - status [ref=e84]: Workspace ready
      - generic [ref=e85]:
        - main [ref=e86]:
          - region "Primary pane" [ref=e89]:
            - generic [ref=e90]:
              - tablist "Primary views" [ref=e91]:
                - tab "analysis.ipynb" [selected] [ref=e92] [cursor=pointer]
              - generic "View actions" [ref=e94]:
                - button "More actions for analysis.ipynb" [ref=e95] [cursor=pointer]: More
                - button "Close analysis.ipynb" [ref=e98] [cursor=pointer]
            - tabpanel "analysis.ipynb" [ref=e101]:
              - generic "analysis.ipynb Notebook reader" [ref=e104]:
                - generic [ref=e105]:
                  - generic [ref=e106]: python · 2 cells
                  - button "Refresh Notebook" [ref=e107] [cursor=pointer]
                - group [ref=e108]:
                  - generic "Saved Notebook · Read only" [ref=e109] [cursor=pointer]
                - generic [ref=e110]:
                  - article "Notebook cell 1" [ref=e111]:
                    - generic [ref=e112]:
                      - strong [ref=e113]: Cell 1
                      - generic [ref=e114]: Markdown
                      - button "View source" [ref=e115] [cursor=pointer]
                    - generic [ref=e116]:
                      - heading "Sample quality review" [level=3] [ref=e117]
                      - paragraph [ref=e118]: Compare the saved code and output before deciding on the threshold.
                      - generic [ref=e119]: Basic saved Markdown · Use Source for exact text and continuation
                  - article "Notebook cell 2" [ref=e120]:
                    - generic [ref=e121]:
                      - strong [ref=e122]: Cell 2
                      - generic [ref=e123]: Code
                    - generic [ref=e124]:
                      - generic "Cell 2 source" [ref=e125]: "# Review sample quality keep = qc[\"mapped_pct\"] >= 80 qc.loc[keep]"
                      - generic [ref=e126]:
                        - button "Select displayed text" [ref=e127] [cursor=pointer]
                        - group [ref=e128]:
                          - generic "Select a text range" [ref=e129] [cursor=pointer]
                    - generic [ref=e130]:
                      - generic [ref=e131]: Saved outputs · 3
                      - button "Collapse outputs" [ref=e132] [cursor=pointer]
                    - region "Cell 2 output 1" [ref=e133]:
                      - generic [ref=e134]: Output 1 · text/plain
                      - paragraph [ref=e135]: Static representation; active alternatives are not rendered.
                      - generic [ref=e136]:
                        - generic "Cell 2 output 1 text" [ref=e137]: sample mapped_pct 0 S01 94.2 2 S03 91.7
                        - generic [ref=e138]:
                          - button "Select displayed text" [ref=e139] [cursor=pointer]
                          - group [ref=e140]:
                            - generic "Select a text range" [ref=e141] [cursor=pointer]
                    - region "Cell 2 output 2" [ref=e142]:
                      - generic [ref=e143]: Output 2 · image/png
                      - generic [ref=e145]:
                        - button "View saved image" [ref=e146] [cursor=pointer]
                        - generic [ref=e147]: 320 × 160 pixels
                    - region "Cell 2 output 3" [ref=e148]:
                      - generic [ref=e149]: Output 3 · Unavailable
                      - paragraph [ref=e150]: No supported saved image or plain-text alternative.
        - separator "Resize chat" [ref=e151]
        - region "Project chat" [ref=e153]:
          - generic [ref=e154]:
            - heading "Chat" [level=2] [ref=e155]
            - button "Project agents" [ref=e156] [cursor=pointer]:
              - text: Agents
              - generic [ref=e159]: "0"
            - button "Collapse chat" [expanded] [ref=e160] [cursor=pointer]
          - generic "Project messages" [ref=e164]:
            - generic [ref=e165]:
              - generic [ref=e166]:
                - strong [ref=e169]: Work together in this Project
                - paragraph [ref=e170]: Choose an agent and start a conversation about your work.
              - generic [ref=e171]: Opened a Project view.
          - generic [ref=e176]:
            - textbox "Message draft" [ref=e177]:
              - /placeholder: Write a message for your agent…
            - generic [ref=e178]:
              - generic [ref=e179]:
                - generic [ref=e180]: Conversation recipient
                - combobox "Conversation recipient" [disabled] [ref=e183]:
                  - option "No agent connected" [selected]
              - button "Send" [disabled] [ref=e184]
            - generic [ref=e187]:
              - generic [ref=e188]: Draft saved locally
              - generic [ref=e189]: Enter to send · Shift+Enter for a new line
```