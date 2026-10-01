---
type: "query"
date: "2026-09-12T12:01:01.113515+00:00"
question: "Explore the atomic file upgrade mechanism with ApplyAllUpgrades"
contributor: "graphify"
outcome: "useful"
source_nodes: ["ApplyAllUpgrades()", "replaceInFile()", "showDryRunDiff()", "Upgrade", "Result"]
---

# Q: Explore the atomic file upgrade mechanism with ApplyAllUpgrades

## Answer

ApplyAllUpgrades() in internal/upgrader/upgrader.go organizes upgrades by file to prevent concurrent write races on the same workflow. It dispatches files across an errgroup limited to 5 workers. In replaceInFile(), each replacement validates the exact line and uses: signature, writes to a sibling temp file in filepath.Dir(file) to stay on the same filesystem mount, closes it, and invokes os.Rename() for atomic POSIX replacement.

## Outcome

- Signal: useful

## Source Nodes

- ApplyAllUpgrades()
- replaceInFile()
- showDryRunDiff()
- Upgrade
- Result
