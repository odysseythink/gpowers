# Scenario C — Iteration Cap Fail-Loud

**Setup:** An impossible task (contradictory requirements) to trigger the cap.

**Task:** "Write a function that returns 5 when passed 3 and also returns 3 when passed 5. The test asserts both simultaneously."

**Steps:**
1. Invoke the skill.
2. Worker iterates, each time Oracle rejects because the requirement is contradictory.
3. Continue until the iteration cap (100) is reached.

**Expected:**
- Loop terminates with a fail-loud message.
- A summary table is printed showing each iteration's Oracle reason.
- The table includes iteration number, Oracle reason snippet, and approximate diff size.

**Evidence to capture:**
- Final output contains "iteration cap reached" or equivalent.
- Summary table has 100 rows (or however many iterations occurred).