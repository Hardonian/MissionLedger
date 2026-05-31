# Policy Model

Current deterministic policy:

Allow:
- read_file
- search_files
- web_search

Escalate:
- terminal
- write_file
- patch
- browser
- browser_click
- unknown tools

Deny:
- secret.read
- send_message

Current truth:
- policy is static code, not user-configurable yet
- policy decisions are appended into proof events
- risky tools can run after mission approval if they are in requested scope
